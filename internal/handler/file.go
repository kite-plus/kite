package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kite-plus/kite/internal/i18n"
	"github.com/kite-plus/kite/internal/middleware"
	"github.com/kite-plus/kite/internal/model"
	"github.com/kite-plus/kite/internal/repo"
	"github.com/kite-plus/kite/internal/service"
)

// FileHandler handles file upload, listing, deletion, and access HTTP requests.
type FileHandler struct {
	fileSvc       *service.FileService
	fileRepo      *repo.FileRepo
	albumRepo     *repo.AlbumRepo
	accessLogRepo *repo.FileAccessLogRepo
}

func NewFileHandler(fileSvc *service.FileService, fileRepo *repo.FileRepo, albumRepo *repo.AlbumRepo, accessLogRepo *repo.FileAccessLogRepo) *FileHandler {
	return &FileHandler{fileSvc: fileSvc, fileRepo: fileRepo, albumRepo: albumRepo, accessLogRepo: accessLogRepo}
}

// Upload handles file uploads via multipart/form-data, compatible with the Lsky v2 upload API.
func (h *FileHandler) Upload(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)

	// Cap the request body before FormFile reads it. Without this guard Gin
	// happily streams a 10 GB body onto /tmp and only errors once the file is
	// already on disk — an easy DoS vector. MaxBytesReader short-circuits the
	// read as soon as the cap is crossed and surfaces as ErrFileTooLarge below.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.fileSvc.MaxUploadBodySize(c.Request.Context()))

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if isRequestBodyTooLarge(err) {
			Fail(c, http.StatusRequestEntityTooLarge, 41300, uploadFileTooLargeMessage)
			return
		}
		BadRequest(c, uploadMissingFileMessage)
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
		BaseURL:  RequestBaseURL(c),
	})
	if err != nil {
		respondUploadError(c, err, false)
		return
	}

	// Respond in the Lsky-compatible shape.
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

// GuestUpload handles anonymous uploads that do not require login.
func (h *FileHandler) GuestUpload(c *gin.Context) {
	// See Upload — same DoS guard applies to anonymous uploads, arguably more
	// so since they don't require an account to abuse.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.fileSvc.MaxUploadBodySize(c.Request.Context()))

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if isRequestBodyTooLarge(err) {
			Fail(c, http.StatusRequestEntityTooLarge, 41300, uploadFileTooLargeMessage)
			return
		}
		BadRequest(c, uploadMissingFileMessage)
		return
	}
	defer file.Close()

	result, err := h.fileSvc.Upload(c.Request.Context(), service.UploadParams{
		UserID:   "guest",
		IsGuest:  true,
		Filename: header.Filename,
		Reader:   file,
		Size:     header.Size,
		BaseURL:  RequestBaseURL(c),
	})
	if err != nil {
		respondUploadError(c, err, true)
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

// List returns files owned by the current user.
// Admins may see all files.
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
		ServerError(c, M(c, i18n.KeyFileListFailed))
		return
	}

	Paged(c, h.EnrichFiles(c.Request.Context(), files, RequestBaseURL(c)), total, page, size)
}

// AdminList returns the site-wide file list unconstrained by user; admin only.
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
		UserID:   c.Query("user_id"), // optional per-user filter
		FileType: c.Query("file_type"),
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: size,
		OrderBy:  c.DefaultQuery("order_by", "created_at"),
		Order:    c.DefaultQuery("order", "DESC"),
	}

	files, total, err := h.fileSvc.ListFiles(c.Request.Context(), params)
	if err != nil {
		ServerError(c, M(c, i18n.KeyFileListFailed))
		return
	}

	Paged(c, h.EnrichFiles(c.Request.Context(), files, RequestBaseURL(c)), total, page, size)
}

// enrichedFile augments model.File with a source URL and fully qualified links.
type enrichedFile struct {
	model.File
	URL       string  `json:"url"`
	ThumbURL  *string `json:"thumb_url,omitempty"`
	SourceURL string  `json:"source_url,omitempty"`
}

func (h *FileHandler) EnrichFiles(ctx context.Context, files []model.File, baseURL string) []enrichedFile {
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
		item.SourceURL = h.fileSvc.GetSourceURL(ctx, &f, baseURL)
		items[i] = item
	}
	return items
}

// AdminDelete deletes any file without checking ownership; admin only.
func (h *FileHandler) AdminDelete(c *gin.Context) {
	id := c.Param("id")
	file, err := h.fileSvc.GetFile(c.Request.Context(), id)
	if err != nil {
		NotFound(c, M(c, i18n.KeyFileNotFound))
		return
	}

	if err := h.fileSvc.DeleteFile(c.Request.Context(), file.ID, file.UserID, "admin"); err != nil {
		ServerError(c, M(c, i18n.KeyFileDeleteFailed))
		return
	}

	Success(c, nil)
}

// Detail returns the details of a single file.
func (h *FileHandler) Detail(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(middleware.ContextKeyUserID)
	file, err := h.fileSvc.GetFile(c.Request.Context(), id)
	if err != nil {
		NotFound(c, M(c, i18n.KeyFileNotFound))
		return
	}
	if file.UserID != userID {
		NotFound(c, M(c, i18n.KeyFileNotFound))
		return
	}
	Success(c, file)
}

// Delete removes a file owned by the current user.
func (h *FileHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(middleware.ContextKeyUserID)

	if err := h.fileSvc.DeleteFile(c.Request.Context(), id, userID, "user"); err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			NotFound(c, M(c, i18n.KeyFileNotFound))
			return
		}
		if errors.Is(err, service.ErrNotFileOwner) {
			Forbidden(c, M(c, i18n.KeyFileNotOwner))
			return
		}
		ServerError(c, M(c, i18n.KeyFileDeleteFailed))
		return
	}

	Success(c, nil)
}

// BatchDelete removes multiple files in one request.
func (h *FileHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeyFileIDsRequired))
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
		Success(c, gin.H{"errors": errs, "deleted": len(req.IDs) - len(errs)})
		return
	}

	Success(c, gin.H{"deleted": len(req.IDs)})
}

// MoveFile moves a file into the given folder by setting album_id; a nil folder_id moves it back to the root.
func (h *FileHandler) MoveFile(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(middleware.ContextKeyUserID)

	var req struct {
		FolderID *string `json:"folder_id"` // null = move to root
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeyErrInvalidRequest))
		return
	}

	file, err := h.fileRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		NotFound(c, M(c, i18n.KeyFileNotFound))
		return
	}
	if file.UserID != userID {
		Forbidden(c, M(c, i18n.KeyFileNotOwner))
		return
	}

	if req.FolderID != nil && *req.FolderID != "" {
		folder, err := h.albumRepo.GetByID(c.Request.Context(), *req.FolderID)
		if err != nil || folder.UserID != userID {
			BadRequest(c, M(c, i18n.KeyFileInvalidTargetFolder))
			return
		}
	}

	if err := h.fileRepo.SetAlbum(c.Request.Context(), id, req.FolderID); err != nil {
		ServerError(c, M(c, i18n.KeyFileMoveFailed))
		return
	}

	Success(c, nil)
}

// ServeImage serves an image over a short link for inline preview.
func (h *FileHandler) ServeImage(c *gin.Context) {
	h.serveFile(c, model.FileTypeImage, false)
}

// ServeVideo serves a video over a short link with Range support.
func (h *FileHandler) ServeVideo(c *gin.Context) {
	h.serveFile(c, model.FileTypeVideo, false)
}

// ServeAudio serves audio over a short link with Range support.
func (h *FileHandler) ServeAudio(c *gin.Context) {
	h.serveFile(c, model.FileTypeAudio, false)
}

// ServeDownload serves a file download over a short link.
func (h *FileHandler) ServeDownload(c *gin.Context) {
	forceDownload := c.Query("dl") == "1"
	h.serveFile(c, "", forceDownload)
}

// ServeThumbnail streams a thumbnail from the storage backend over a short link.
// If the thumbnail is missing (never written, deleted out-of-band, or pointed at by
// an orphaned thumb_url), it is regenerated from the source file on demand so the
// next request can be served directly from storage.
func (h *FileHandler) ServeThumbnail(c *gin.Context) {
	hash := c.Param("hash")

	file, err := h.fileRepo.GetByHashPrefix(c.Request.Context(), hash)
	if err != nil {
		NotFound(c, M(c, i18n.KeyFileNotFound))
		return
	}
	if file.FileType != model.FileTypeImage {
		NotFound(c, M(c, i18n.KeyFileThumbnailNotFound))
		return
	}

	reader, size, err := h.fileSvc.GetThumbContent(c.Request.Context(), file)
	if err != nil {
		reader, size, err = h.fileSvc.RegenerateThumbnail(c.Request.Context(), file)
		if err != nil {
			NotFound(c, M(c, i18n.KeyFileThumbnailNotFound))
			return
		}
	}
	defer reader.Close()

	// Thumbnail format mirrors the source: PNG/WebP/GIF sources keep their alpha and are
	// encoded as PNG, everything else becomes JPEG. This must match ImageService.GenerateThumbnail
	// or the Content-Type will not agree with the bytes and the browser may fail to decode a PNG as JPEG.
	thumbMime := service.ThumbnailMimeFor(file.MimeType)
	c.Header("Content-Type", thumbMime)
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("ETag", file.HashMD5+"-thumb")
	c.DataFromReader(http.StatusOK, size, thumbMime, reader, nil)
	h.logAccess(file.ID, file.UserID, size)
}

// logAccess records a file access asynchronously; failures do not affect the request.
// userID is the owner of the file (empty or "guest" for anonymous uploads) and is used
// to aggregate access counts per user.
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

	// Look up the file matching the hash prefix.
	file, err := h.findFileByHash(c, hash)
	if err != nil {
		NotFound(c, M(c, i18n.KeyFileNotFound))
		return
	}

	// Optional WebP variant: when the canonical file was uploaded with
	// the keep_original=true sidecar policy, a `<storage_key>.webp`
	// neighbour exists and the client can request it via ?fmt=webp.
	// Falls through transparently to the canonical file when no sidecar
	// is present, so the same URL always works for any image.
	wantWebP := !forceDownload && c.Query("dl") != "1" &&
		c.Query("fmt") == "webp" && file.FileType == model.FileTypeImage

	var reader io.ReadCloser
	var size int64
	servedFromSidecar := false
	if wantWebP {
		if r, sz, sErr := h.fileSvc.GetWebPSidecarContent(c.Request.Context(), file); sErr == nil {
			reader = r
			size = sz
			servedFromSidecar = true
		}
	}
	if reader == nil {
		reader, size, err = h.fileSvc.GetFileContent(c.Request.Context(), file)
		if err != nil {
			ServerError(c, M(c, i18n.KeyFileReadFailed))
			return
		}
	}
	defer reader.Close()

	contentType := "application/octet-stream"
	if forceDownload || c.Query("dl") == "1" {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
	} else {
		contentType = file.MimeType
		if servedFromSidecar {
			contentType = "image/webp"
		}
		switch file.FileType {
		case model.FileTypeImage:
			c.Header("Content-Disposition", "inline")
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
			etag := file.HashMD5
			if servedFromSidecar {
				etag = file.HashMD5 + ".webp"
			}
			c.Header("ETag", etag)
		case model.FileTypeVideo, model.FileTypeAudio:
			c.Header("Accept-Ranges", "bytes")
		default:
			// `?inline=1` opts the share page's iframe into inline rendering
			// (e.g. PDF preview). Without it the default disposition is still
			// attachment so direct /f/<hash> links keep their download
			// behaviour.
			if c.Query("inline") == "1" {
				c.Header("Content-Disposition", "inline")
			} else {
				c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
			}
		}
	}

	c.DataFromReader(http.StatusOK, size, contentType, reader, nil)
	h.logAccess(file.ID, file.UserID, size)
}

func (h *FileHandler) findFileByHash(c *gin.Context, hash string) (*model.File, error) {
	return h.fileRepo.GetByHashPrefix(c.Request.Context(), hash)
}

func RequestBaseURL(c *gin.Context) string {
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if c.Request.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + c.Request.Host
}

const (
	uploadMissingFileMessage    = "请选择要上传的文件"
	uploadFileTooLargeMessage   = "文件超过单文件大小限制，请压缩后重试"
	uploadFileTypeDeniedMessage = "该文件类型暂不支持上传"
	uploadStorageFullMessage    = "存储空间不足，暂时无法上传"
	uploadSaveFailedMessage     = "文件保存失败，请稍后重试"
	guestUploadFailedMessage    = "上传处理失败，请稍后重试"
)

// isRequestBodyTooLarge reports whether FormFile failed because MaxBytesReader
// tripped the per-request cap. Both the typed error (Go 1.18+) and the legacy
// "http: request body too large" string are checked so the helper survives
// wrappers that swallow the concrete type.
func isRequestBodyTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return true
	}
	return strings.Contains(err.Error(), "http: request body too large")
}

func respondUploadError(c *gin.Context, err error, isGuest bool) {
	switch {
	case errors.Is(err, service.ErrFileTooLarge):
		Fail(c, http.StatusRequestEntityTooLarge, 41300, uploadFileTooLargeMessage)
	case errors.Is(err, service.ErrFileTypeDenied):
		Fail(c, http.StatusUnsupportedMediaType, 41500, uploadFileTypeDeniedMessage)
	case errors.Is(err, service.ErrStorageFull):
		Fail(c, http.StatusInsufficientStorage, 50700, uploadStorageFullMessage)
	default:
		if isGuest {
			ServerError(c, guestUploadFailedMessage)
			return
		}
		ServerError(c, uploadSaveFailedMessage)
	}
}

func fileExtension(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}
	return ""
}
