package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/amigoer/kite/internal/api/middleware"
	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
)

// FileHandler 文件上传、查询、删除、访问的 HTTP 处理器。
type FileHandler struct {
	fileSvc  *service.FileService
	fileRepo *repo.FileRepo
}

func NewFileHandler(fileSvc *service.FileService, fileRepo *repo.FileRepo) *FileHandler {
	return &FileHandler{fileSvc: fileSvc, fileRepo: fileRepo}
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

	paged(c, files, total, page, size)
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

	paged(c, files, total, page, size)
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
	file, err := h.fileSvc.GetFile(c.Request.Context(), id)
	if err != nil {
		notFound(c, "file not found")
		return
	}
	success(c, file)
}

// Delete 删除文件。
func (h *FileHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(middleware.ContextKeyUserID)
	role := c.GetString(middleware.ContextKeyRole)

	if err := h.fileSvc.DeleteFile(c.Request.Context(), id, userID, role); err != nil {
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
	role := c.GetString(middleware.ContextKeyRole)

	var errs []string
	for _, id := range req.IDs {
		if err := h.fileSvc.DeleteFile(c.Request.Context(), id, userID, role); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", id, err.Error()))
		}
	}

	if len(errs) > 0 {
		success(c, gin.H{"errors": errs, "deleted": len(req.IDs) - len(errs)})
		return
	}

	success(c, gin.H{"deleted": len(req.IDs)})
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

// ServeThumbnail 通过短链提供缩略图访问。
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

	c.Redirect(http.StatusFound, *file.ThumbURL)
}

func (h *FileHandler) serveFile(c *gin.Context, expectedType string, forceDownload bool) {
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

	// 根据文件类型和参数设置响应头
	if forceDownload || c.Query("dl") == "1" {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
		c.Header("Content-Type", "application/octet-stream")
	} else {
		c.Header("Content-Type", file.MimeType)
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

	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Status(http.StatusOK)
	c.Stream(func(w io.Writer) bool {
		_, err := io.Copy(w, reader)
		return err == nil
	})
}

func (h *FileHandler) findFileByHash(c *gin.Context, hash string) (*model.File, error) {
	return h.fileRepo.GetByHashPrefix(c.Request.Context(), hash)
}

func fileExtension(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}
	return ""
}
