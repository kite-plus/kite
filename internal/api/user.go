package api

import (
	"strconv"
	"strings"
	"time"

	"github.com/amigoer/kite/internal/api/middleware"
	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// UserHandler 用户管理的 HTTP 处理器（管理员专用）。
type UserHandler struct {
	userRepo      *repo.UserRepo
	fileRepo      *repo.FileRepo
	accessLogRepo *repo.FileAccessLogRepo
	authSvc       *service.AuthService
}

func NewUserHandler(userRepo *repo.UserRepo, fileRepo *repo.FileRepo, accessLogRepo *repo.FileAccessLogRepo, authSvc *service.AuthService) *UserHandler {
	return &UserHandler{userRepo: userRepo, fileRepo: fileRepo, accessLogRepo: accessLogRepo, authSvc: authSvc}
}

// List 获取所有用户列表。
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
		serverError(c, "failed to list users")
		return
	}

	paged(c, users, total, page, size)
}

type createUserRequest struct {
	Username     string `json:"username" binding:"required,min=3,max=32"`
	Nickname     string `json:"nickname" binding:"max=32"`
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=6,max=64"`
	Role         string `json:"role" binding:"required,oneof=admin user"`
	StorageLimit *int64 `json:"storage_limit"`
}

// Create 管理员创建用户。
func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid user data: "+err.Error())
		return
	}

	var user interface{}
	var err error

	if req.Role == "admin" {
		user, err = h.authSvc.CreateAdminUser(c.Request.Context(), req.Username, req.Email, req.Password, false)
	} else {
		user, err = h.authSvc.Register(c.Request.Context(), req.Username, req.Email, req.Password)
	}

	if err != nil {
		serverError(c, "failed to create user: "+err.Error())
		return
	}

	if createdUser, ok := user.(*model.User); ok {
		nickname := strings.TrimSpace(req.Nickname)
		if nickname != "" {
			createdUser.Nickname = &nickname
			if err := h.userRepo.Update(c.Request.Context(), createdUser); err != nil {
				serverError(c, "failed to save user nickname")
				return
			}
		}
	}

	created(c, user)
}

type updateUserRequest struct {
	Nickname     *string `json:"nickname" binding:"omitempty,max=32"`
	Email        *string `json:"email" binding:"omitempty,email"`
	Password     *string `json:"password" binding:"omitempty,min=6,max=64"`
	Role         *string `json:"role" binding:"omitempty,oneof=admin user"`
	IsActive     *bool   `json:"is_active"`
	StorageLimit *int64  `json:"storage_limit"`
}

// Update 管理员更新用户信息。
func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")

	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		notFound(c, "user not found")
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid user data: "+err.Error())
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
			serverError(c, "failed to validate user email")
			return
		}
		if conflict {
			fail(c, 409, 40900, service.ErrUserExists.Error())
			return
		}
		user.Email = *req.Email
	}
	if req.Password != nil && strings.TrimSpace(*req.Password) != "" {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if hashErr != nil {
			serverError(c, "failed to hash password")
			return
		}
		user.PasswordHash = string(hash)
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.StorageLimit != nil {
		user.StorageLimit = *req.StorageLimit
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		serverError(c, "failed to update user")
		return
	}

	success(c, user)
}

// Delete 管理员删除用户（禁用）。
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.userRepo.Delete(c.Request.Context(), id); err != nil {
		serverError(c, "failed to delete user")
		return
	}

	success(c, nil)
}

// Stats 获取当前登录用户的使用统计（仅自己的数据）。
// 管理员若需全站数据，应调用 AdminStats。
func (h *UserHandler) Stats(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)
	stats, err := h.fileRepo.GetUserStats(c.Request.Context(), userID)
	if err != nil {
		serverError(c, "failed to get stats")
		return
	}

	success(c, gin.H{
		"total_files": stats.TotalFiles,
		"total_size":  stats.TotalSize,
		"images":      stats.ImageCount,
		"videos":      stats.VideoCount,
		"audios":      stats.AudioCount,
		"others":      stats.OtherCount,
	})
}

// AdminStats 获取全站使用统计（仅管理员可调用）。
func (h *UserHandler) AdminStats(c *gin.Context) {
	stats, err := h.fileRepo.GetStats(c.Request.Context())
	if err != nil {
		serverError(c, "failed to get stats")
		return
	}

	userCount, _ := h.userRepo.Count(c.Request.Context())

	success(c, gin.H{
		"users":       userCount,
		"total_files": stats.TotalFiles,
		"total_size":  stats.TotalSize,
		"images":      stats.ImageCount,
		"videos":      stats.VideoCount,
		"audios":      stats.AudioCount,
		"others":      stats.OtherCount,
	})
}

// DailyStats 获取当前用户按天分组的上传数、访问数、带宽统计。
// 返回连续 N 天（默认 7）的时间序列，缺失日期补零。
func (h *UserHandler) DailyStats(c *gin.Context) {
	h.dailyStats(c, c.GetString(middleware.ContextKeyUserID))
}

// AdminDailyStats 获取全站按天分组的上传数、访问数、带宽统计（仅管理员可调用）。
func (h *UserHandler) AdminDailyStats(c *gin.Context) {
	h.dailyStats(c, "")
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
		serverError(c, "failed to get upload stats")
		return
	}
	accesses, err := h.accessLogRepo.GetDailyAccessStats(c.Request.Context(), userID, start, end)
	if err != nil {
		serverError(c, "failed to get access stats")
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

	success(c, gin.H{"days": series})
}
