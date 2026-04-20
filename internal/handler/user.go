package handler

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/amigoer/kite/internal/middleware"
	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// UserHandler handles admin-only user management HTTP requests.
type UserHandler struct {
	userRepo      *repo.UserRepo
	fileRepo      *repo.FileRepo
	accessLogRepo *repo.FileAccessLogRepo
	authSvc       *service.AuthService
}

func NewUserHandler(userRepo *repo.UserRepo, fileRepo *repo.FileRepo, accessLogRepo *repo.FileAccessLogRepo, authSvc *service.AuthService) *UserHandler {
	return &UserHandler{userRepo: userRepo, fileRepo: fileRepo, accessLogRepo: accessLogRepo, authSvc: authSvc}
}

// List returns a paginated list of all users.
func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	users, total, err := h.userRepo.List(c.Request.Context(), page, size)
	if err != nil {
		ServerError(c, "failed to list users")
		return
	}

	Paged(c, users, total, page, size)
}

type createUserRequest struct {
	Username     string `json:"username" binding:"required,min=3,max=32"`
	Nickname     string `json:"nickname" binding:"max=32"`
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=6,max=64"`
	Role         string `json:"role" binding:"required,oneof=admin user"`
	StorageLimit *int64 `json:"storage_limit"`
}

// Create creates a new user as an admin.
func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid user data: "+err.Error())
		return
	}

	var user interface{}
	var err error

	if req.Role == "admin" {
		user, err = h.authSvc.CreateAdminUser(c.Request.Context(), req.Username, req.Email, req.Password, false)
	} else {
		user, err = h.authSvc.CreateStandardUser(c.Request.Context(), req.Username, req.Email, req.Password)
	}

	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			Fail(c, 409, 40900, err.Error())
			return
		}
		ServerError(c, "failed to create user: "+err.Error())
		return
	}

	if createdUser, ok := user.(*model.User); ok {
		nickname := strings.TrimSpace(req.Nickname)
		if nickname != "" {
			createdUser.Nickname = &nickname
			if err := h.userRepo.Update(c.Request.Context(), createdUser); err != nil {
				ServerError(c, "failed to save user nickname")
				return
			}
		}
	}

	Created(c, user)
}

type updateUserRequest struct {
	Nickname     *string `json:"nickname" binding:"omitempty,max=32"`
	Email        *string `json:"email" binding:"omitempty,email"`
	Password     *string `json:"password" binding:"omitempty,min=6,max=64"`
	Role         *string `json:"role" binding:"omitempty,oneof=admin user"`
	IsActive     *bool   `json:"is_active"`
	StorageLimit *int64  `json:"storage_limit"`
}

// Update updates a user's profile as an admin.
func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")

	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "user not found")
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid user data: "+err.Error())
		return
	}

	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Nickname != nil {
		nickname := strings.TrimSpace(*req.Nickname)
		if nickname == "" {
			user.Nickname = nil
		} else {
			user.Nickname = &nickname
		}
	}
	if req.Email != nil {
		conflict, conflictErr := h.userRepo.ExistsByUsernameOrEmailExcept(c.Request.Context(), user.Username, *req.Email, user.ID)
		if conflictErr != nil {
			ServerError(c, "failed to validate user email")
			return
		}
		if conflict {
			Fail(c, 409, 40900, service.ErrUserExists.Error())
			return
		}
		user.Email = *req.Email
	}
	if req.Password != nil && strings.TrimSpace(*req.Password) != "" {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if hashErr != nil {
			ServerError(c, "failed to hash password")
			return
		}
		user.PasswordHash = string(hash)
		user.HasLocalPassword = true
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.StorageLimit != nil {
		user.StorageLimit = *req.StorageLimit
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		ServerError(c, "failed to update user")
		return
	}

	Success(c, user)
}

// Delete deactivates a user as an admin.
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.userRepo.Delete(c.Request.Context(), id); err != nil {
		ServerError(c, "failed to delete user")
		return
	}

	Success(c, nil)
}

// Stats returns usage statistics scoped to the current user.
// Admins wanting site-wide data should call AdminStats instead.
func (h *UserHandler) Stats(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)
	stats, err := h.fileRepo.GetUserStats(c.Request.Context(), userID)
	if err != nil {
		ServerError(c, "failed to get stats")
		return
	}

	Success(c, gin.H{
		"total_files": stats.TotalFiles,
		"total_size":  stats.TotalSize,
		"images":      stats.ImageCount,
		"videos":      stats.VideoCount,
		"audios":      stats.AudioCount,
		"others":      stats.OtherCount,
		"images_size": stats.ImageSize,
		"videos_size": stats.VideoSize,
		"audios_size": stats.AudioSize,
		"others_size": stats.OtherSize,
	})
}

// AdminStats returns site-wide usage statistics; admin only.
func (h *UserHandler) AdminStats(c *gin.Context) {
	stats, err := h.fileRepo.GetStats(c.Request.Context())
	if err != nil {
		ServerError(c, "failed to get stats")
		return
	}

	userCount, _ := h.userRepo.Count(c.Request.Context())

	Success(c, gin.H{
		"users":       userCount,
		"total_files": stats.TotalFiles,
		"total_size":  stats.TotalSize,
		"images":      stats.ImageCount,
		"videos":      stats.VideoCount,
		"audios":      stats.AudioCount,
		"others":      stats.OtherCount,
		"images_size": stats.ImageSize,
		"videos_size": stats.VideoSize,
		"audios_size": stats.AudioSize,
		"others_size": stats.OtherSize,
	})
}

// DailyStats returns per-day upload counts, access counts, and bandwidth for the current user.
// The response is a continuous N-day series (default 7) with zero-filled gaps.
func (h *UserHandler) DailyStats(c *gin.Context) {
	h.dailyStats(c, c.GetString(middleware.ContextKeyUserID))
}

// HeatmapStats returns a weekday-by-hour activity heatmap for the current user over the last N weeks.
func (h *UserHandler) HeatmapStats(c *gin.Context) {
	h.heatmapStats(c, c.GetString(middleware.ContextKeyUserID))
}

// AdminDailyStats returns site-wide per-day upload, access, and bandwidth stats; admin only.
func (h *UserHandler) AdminDailyStats(c *gin.Context) {
	h.dailyStats(c, "")
}

// AdminHeatmapStats returns a site-wide weekday-by-hour heatmap over the last N weeks; admin only.
func (h *UserHandler) AdminHeatmapStats(c *gin.Context) {
	h.heatmapStats(c, "")
}

func (h *UserHandler) dailyStats(c *gin.Context, userID string) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days < 1 || days > 90 {
		days = 7
	}

	now := time.Now()
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	start := end.AddDate(0, 0, -days)

	uploads, err := h.fileRepo.GetDailyUploadStats(c.Request.Context(), userID, start, end)
	if err != nil {
		ServerError(c, "failed to get upload stats")
		return
	}
	accesses, err := h.accessLogRepo.GetDailyAccessStats(c.Request.Context(), userID, start, end)
	if err != nil {
		ServerError(c, "failed to get access stats")
		return
	}

	uploadMap := make(map[string]int64, len(uploads))
	for _, u := range uploads {
		uploadMap[u.Day] = u.UploadCount
	}
	accessMap := make(map[string]int64, len(accesses))
	bytesMap := make(map[string]int64, len(accesses))
	for _, a := range accesses {
		accessMap[a.Day] = a.AccessCount
		bytesMap[a.Day] = a.BytesServed
	}

	type daily struct {
		Day         string `json:"day"`
		Uploads     int64  `json:"uploads"`
		Accesses    int64  `json:"accesses"`
		BytesServed int64  `json:"bytes_served"`
	}

	series := make([]daily, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		series = append(series, daily{
			Day:         key,
			Uploads:     uploadMap[key],
			Accesses:    accessMap[key],
			BytesServed: bytesMap[key],
		})
	}

	Success(c, gin.H{"days": series})
}

func (h *UserHandler) heatmapStats(c *gin.Context, userID string) {
	weeks, _ := strconv.Atoi(c.DefaultQuery("weeks", "12"))
	if weeks < 1 || weeks > 52 {
		weeks = 12
	}

	now := time.Now()
	end := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location()).Add(time.Hour)
	start := end.AddDate(0, 0, -weeks*7)

	uploads, err := h.fileRepo.GetHourlyUploadHeatmapStats(c.Request.Context(), userID, start, end)
	if err != nil {
		ServerError(c, "failed to get upload heatmap stats")
		return
	}
	accesses, err := h.accessLogRepo.GetHourlyAccessHeatmapStats(c.Request.Context(), userID, start, end)
	if err != nil {
		ServerError(c, "failed to get access heatmap stats")
		return
	}

	grid := make([][]int64, 7)
	for i := range grid {
		grid[i] = make([]int64, 24)
	}

	for _, u := range uploads {
		row, col, ok := normalizeHeatmapCell(u.Weekday, u.Hour)
		if ok {
			grid[row][col] += u.Count
		}
	}
	for _, a := range accesses {
		row, col, ok := normalizeHeatmapCell(a.Weekday, a.Hour)
		if ok {
			grid[row][col] += a.Count
		}
	}

	Success(c, gin.H{
		"weeks": weeks,
		"grid":  grid,
	})
}

func normalizeHeatmapCell(weekday int, hour int) (row int, col int, ok bool) {
	if weekday < 0 || weekday > 6 || hour < 0 || hour > 23 {
		return 0, 0, false
	}
	// SQLite weekday: Sunday=0 ... Saturday=6, UI row: Monday=0 ... Sunday=6.
	row = (weekday + 6) % 7
	return row, hour, true
}
