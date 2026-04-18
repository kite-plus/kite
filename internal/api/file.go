package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/amigoer/kite/internal/api/middleware"
	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FileHandler 文件上传、查询、删除、访问的 HTTP 处理器。
type FileHandler struct {
	fileSvc       *service.FileService
	fileRepo      *repo.FileRepo
	albumRepo     *repo.AlbumRepo
	accessLogRepo *repo.FileAccessLogRepo
}

func NewFileHandler(fileSvc *service.FileService, fileRepo *repo.FileRepo, albumRepo *repo.AlbumRepo, accessLogRepo *repo.FileAccessLogRepo) *FileHandler {
	return &FileHandler{fileSvc: fileSvc, fileRepo: fileRepo, albumRepo: albumRepo, accessLogRepo: accessLogRepo}
}

// Upload 处理文件上传。
// 支持 multipart/form-data，兼容兰空 v2 上传接口。
func (h *FileHandler) Upload(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		badRequest(c, "file is required")
		return
	}
	defer file.Close()

	var albumID *string
	if aid := c.PostForm("album_id"); aid != "" {
		albumID = &aid
	}

	result, err := h.fileSvc.Upload(c.Request.Context(), service.UploadParams{
		UserID:   userID,
		AlbumID:  albumID,
		Filename: header.Filename,
		Reader:   file,
		Size:     header.Size,
		BaseURL:  requestBaseURL(c),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFileTooLarge):
			fail(c, http.StatusRequestEntityTooLarge, 41300, err.Error())
		case errors.Is(err, service.ErrFileTypeDenied):
			fail(c, http.StatusUnsupportedMediaType, 41500, err.Error())
		case errors.Is(err, service.ErrStorageFull):
			fail(c, http.StatusInsufficientStorage, 50700, err.Error())
		default:
			serverError(c, "upload failed")
		}
		return
	}

	// 兰空兼容格式响应
	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"message": "success",
		"data": gin.H{
			"key":         result.File.StorageKey,
			"name":        result.File.OriginalName,
			"origin_name": result.File.OriginalName,
			"size":        result.File.SizeBytes,
			"mimetype":    result.File.MimeType,
			"extension":   fileExtension(result.File.OriginalName),
			"md5":         result.File.HashMD5,
			"links":       result.Links,
		},
	})
}

// GuestUpload 处理游客文件上传（无需登录）。
func (h *FileHandler) GuestUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		badRequest(c, "file is required")
		return
	}
	defer file.Close()

	result, err := h.fileSvc.Upload(c.Request.Context(), service.UploadParams{
		UserID:   "guest",
		IsGuest:  true,
		Filename: header.Filename,
		Reader:   file,
		Size:     header.Size,
		BaseURL:  requestBaseURL(c),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFileTooLarge):
			fail(c, http.StatusRequestEntityTooLarge, 41300, err.Error())
		case errors.Is(err, service.ErrFileTypeDenied):
			fail(c, http.StatusUnsupportedMediaType, 41500, err.Error())
		default:
			serverError(c, "upload failed")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"message": "success",
		"data": gin.H{
			"key":         result.File.StorageKey,
			"name":        result.File.OriginalName,
			"origin_name": result.File.OriginalName,
			"size":        result.File.SizeBytes,
			"mimetype":    result.File.MimeType,
			"extension":   fileExtension(result.File.OriginalName),
			"md5":         result.File.HashMD5,
			"links":       result.Links,
		},
	})
}

// List 获取当前用户的文件列表。
// 管理员可以查看全部文件。
func (h *FileHandler) List(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	params := repo.FileListParams{
		UserID:   userID,
		NoAlbum:  c.Query("no_album") == "true",
		AlbumID:  c.Query("album_id"),
		FileType: c.Query("file_type"),
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: size,
		OrderBy:  c.DefaultQuery("order_by", "created_at"),
		Order:    c.DefaultQuery("order", "DESC"),
	}

	files, total, err := h.fileSvc.ListFiles(c.Request.Context(), params)
	if err != nil {
		serverError(c, "failed to list files")
		return
	}

	paged(c, h.enrichFiles(files, requestBaseURL(c)), total, page, size)
}

// AdminList 管理员查看全站文件列表（不限制用户）。
func (h *FileHandler) AdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	params := repo.FileListParams{
		UserID:   c.Query("user_id"), // 可选按用户筛选
		FileType: c.Query("file_type"),
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: size,
		OrderBy:  c.DefaultQuery("order_by", "created_at"),
		Order:    c.DefaultQuery("order", "DESC"),
	}

	files, total, err := h.fileSvc.ListFiles(c.Request.Context(), params)
	if err != nil {
		serverError(c, "failed to list files")
		return
	}

	paged(c, h.enrichFiles(files, requestBaseURL(c)), total, page, size)
}

// enrichedFile 在 model.File 基础上添加源站 URL 和完整链接。
type enrichedFile struct {
	model.File
	URL       string  `json:"url"`
	ThumbURL  *string `json:"thumb_url,omitempty"`
	SourceURL string  `json:"source_url,omitempty"`
}

func (h *FileHandler) enrichFiles(files []model.File, baseURL string) []enrichedFile {
	base := strings.TrimRight(baseURL, "/")
	items := make([]enrichedFile, len(files))
	for i := range files {
		f := files[i]
		item := enrichedFile{File: f, URL: f.URL, ThumbURL: f.ThumbURL}
		if base != "" && !strings.HasPrefix(item.URL, "http") {
			item.URL = base + item.URL
		}
		if item.ThumbURL != nil && !strings.HasPrefix(*item.ThumbURL, "http") {
			thumbURL := base + *item.ThumbURL
			item.ThumbURL = &thumbURL
		}
		item.SourceURL = h.fileSvc.GetSourceURL(&f, baseURL)
		items[i] = item
	}
	return items
}

// AdminDelete 管理员删除任意文件（不检查所属用户）。
func (h *FileHandler) AdminDelete(c *gin.Context) {
	id := c.Param("id")
	file, err := h.fileSvc.GetFile(c.Request.Context(), id)
	if err != nil {
		notFound(c, "file not found")
		return
	}

	if err := h.fileSvc.DeleteFile(c.Request.Context(), file.ID, file.UserID, "admin"); err != nil {
		serverError(c, "failed to delete file")
		return
	}

	success(c, nil)
}

// Detail 获取文件详情。
func (h *FileHandler) Detail(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(middleware.ContextKeyUserID)
	file, err := h.fileSvc.GetFile(c.Request.Context(), id)
	if err != nil {
		notFound(c, "file not found")
		return
	}
	if file.UserID != userID {
		notFound(c, "file not found")
		return
	}
	success(c, file)
}

// Delete 删除文件。
func (h *FileHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(middleware.ContextKeyUserID)

	if err := h.fileSvc.DeleteFile(c.Request.Context(), id, userID, "user"); err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			notFound(c, "file not found")
			return
		}
		if errors.Is(err, service.ErrNotFileOwner) {
			forbidden(c, "not the owner of this file")
			return
		}
		serverError(c, "failed to delete file")
		return
	}

	success(c, nil)
}

// BatchDelete 批量删除文件。
func (h *FileHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "ids is required")
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)

	var errs []string
	for _, id := range req.IDs {
		if err := h.fileSvc.DeleteFile(c.Request.Context(), id, userID, "user"); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", id, err.Error()))
		}
	}

	if len(errs) > 0 {
		success(c, gin.H{"errors": errs, "deleted": len(req.IDs) - len(errs)})
		return
	}

	success(c, gin.H{"deleted": len(req.IDs)})
}

// MoveFile 移动文件到指定文件夹（设置 album_id）。folder_id 为 null 时移出所有文件夹。
func (h *FileHandler) MoveFile(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(middleware.ContextKeyUserID)

	var req struct {
		FolderID *string `json:"folder_id"` // null = 移到根目录
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid request")
		return
	}

	file, err := h.fileRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		notFound(c, "file not found")
		return
	}
	if file.UserID != userID {
		forbidden(c, "not the owner of this file")
		return
	}

	if req.FolderID != nil && *req.FolderID != "" {
		folder, err := h.albumRepo.GetByID(c.Request.Context(), *req.FolderID)
		if err != nil || folder.UserID != userID {
			badRequest(c, "invalid target folder")
			return
		}
	}

	if err := h.fileRepo.SetAlbum(c.Request.Context(), id, req.FolderID); err != nil {
		serverError(c, "failed to move file")
		return
	}

	success(c, nil)
}

// ServeImage 通过短链提供图片访问（内联预览）。
func (h *FileHandler) ServeImage(c *gin.Context) {
	h.serveFile(c, model.FileTypeImage, false)
}

// ServeVideo 通过短链提供视频访问（支持 Range）。
func (h *FileHandler) ServeVideo(c *gin.Context) {
	h.serveFile(c, model.FileTypeVideo, false)
}

// ServeAudio 通过短链提供音频访问（支持 Range）。
func (h *FileHandler) ServeAudio(c *gin.Context) {
	h.serveFile(c, model.FileTypeAudio, false)
}

// ServeDownload 通过短链提供文件下载。
func (h *FileHandler) ServeDownload(c *gin.Context) {
	forceDownload := c.Query("dl") == "1"
	h.serveFile(c, "", forceDownload)
}

// ServeThumbnail 通过短链从存储后端流式输出缩略图。
func (h *FileHandler) ServeThumbnail(c *gin.Context) {
	hash := c.Param("hash")

	file, err := h.fileRepo.GetByHashPrefix(c.Request.Context(), hash)
	if err != nil {
		notFound(c, "file not found")
		return
	}
	if file.ThumbURL == nil {
		notFound(c, "thumbnail not found")
		return
	}

	reader, size, err := h.fileSvc.GetThumbContent(c.Request.Context(), file)
	if err != nil {
		notFound(c, "thumbnail not found")
		return
	}
	defer reader.Close()

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("ETag", file.HashMD5+"-thumb")
	c.DataFromReader(http.StatusOK, size, "image/jpeg", reader, nil)
	h.logAccess(file.ID, file.UserID, size)
}

// logAccess 异步记录一次文件访问；失败不影响请求。
// userID 为文件所有者（游客文件为 "guest" 或空），用于按用户维度统计访问量。
func (h *FileHandler) logAccess(fileID, userID string, bytes int64) {
	if h.accessLogRepo == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = h.accessLogRepo.Create(ctx, &model.FileAccessLog{
			ID:          uuid.New().String(),
			FileID:      fileID,
			UserID:      userID,
			BytesServed: bytes,
		})
	}()
}

func (h *FileHandler) serveFile(c *gin.Context, _ string, forceDownload bool) {
	hash := c.Param("hash")

	// 查找匹配 hash 前缀的文件
	file, err := h.findFileByHash(c, hash)
	if err != nil {
		notFound(c, "file not found")
		return
	}

	// 获取文件内容
	reader, size, err := h.fileSvc.GetFileContent(c.Request.Context(), file)
	if err != nil {
		serverError(c, "failed to read file")
		return
	}
	defer reader.Close()

	contentType := "application/octet-stream"
	if forceDownload || c.Query("dl") == "1" {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
	} else {
		contentType = file.MimeType
		switch file.FileType {
		case model.FileTypeImage:
			c.Header("Content-Disposition", "inline")
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
			c.Header("ETag", file.HashMD5)
		case model.FileTypeVideo, model.FileTypeAudio:
			c.Header("Accept-Ranges", "bytes")
		default:
			c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
		}
	}

	c.DataFromReader(http.StatusOK, size, contentType, reader, nil)
	h.logAccess(file.ID, file.UserID, size)
}

func (h *FileHandler) findFileByHash(c *gin.Context, hash string) (*model.File, error) {
	return h.fileRepo.GetByHashPrefix(c.Request.Context(), hash)
}

func requestBaseURL(c *gin.Context) string {
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if c.Request.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + c.Request.Host
}

func fileExtension(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}
	return ""
}
