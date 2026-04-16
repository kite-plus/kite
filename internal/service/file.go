package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/amigoer/kite/internal/config"
	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/storage"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
)

var (
	ErrFileTooLarge   = errors.New("file exceeds maximum size limit")
	ErrFileTypeDenied = errors.New("file type is not allowed")
	ErrStorageFull    = errors.New("storage quota exceeded")
	ErrFileNotFound   = errors.New("file not found")
	ErrNotFileOwner   = errors.New("not the owner of this file")
	ErrDuplicateFile  = errors.New("file already exists")
)

// FileService 文件处理核心业务逻辑。
type FileService struct {
	fileRepo   *repo.FileRepo
	userRepo   *repo.UserRepo
	storageMgr *storage.Manager
	imageSvc   *ImageService
	cfg        config.UploadConfig
}

func NewFileService(
	fileRepo *repo.FileRepo,
	userRepo *repo.UserRepo,
	storageMgr *storage.Manager,
	imageSvc *ImageService,
	cfg config.UploadConfig,
) *FileService {
	return &FileService{
		fileRepo:   fileRepo,
		userRepo:   userRepo,
		storageMgr: storageMgr,
		imageSvc:   imageSvc,
		cfg:        cfg,
	}
}

// UploadParams 上传参数。
type UploadParams struct {
	UserID   string
	AlbumID  *string
	Filename string
	Reader   io.Reader
	Size     int64
	IsGuest  bool   // 游客上传模式，跳过用户配额检查
	BaseURL  string // 请求来源（如 "https://demo.kite.plus"），用于生成完整链接
}

// UploadResult 上传结果。
type UploadResult struct {
	File  *model.File
	Links FileLinks
}

// FileLinks 各种格式的访问链接（兼容兰空格式）。
type FileLinks struct {
	URL              string `json:"url"`                  // 短链接，形如 /i/{hash}
	SourceURL        string `json:"source_url,omitempty"` // 原始存储 URL，指向存储后端中的真实路径
	HTML             string `json:"html"`
	BBCode           string `json:"bbcode"`
	Markdown         string `json:"markdown"`
	MarkdownWithLink string `json:"markdown_with_link"`
	ThumbnailURL     string `json:"thumbnail_url,omitempty"`
}

// Upload 处理文件上传的完整流程。
func (s *FileService) Upload(ctx context.Context, params UploadParams) (*UploadResult, error) {
	// 1. 检查文件大小
	if params.Size > s.cfg.MaxFileSize {
		return nil, ErrFileTooLarge
	}

	// 2. 检查用户存储配额（游客模式跳过）
	if !params.IsGuest {
		user, err := s.userRepo.GetByID(ctx, params.UserID)
		if err != nil {
			return nil, fmt.Errorf("upload get user: %w", err)
		}
		if !user.HasStorageSpace(params.Size) {
			return nil, ErrStorageFull
		}
	}

	// 3. 读取全部内容到内存以进行 MIME 检测和 MD5 计算
	data, err := io.ReadAll(params.Reader)
	if err != nil {
		return nil, fmt.Errorf("upload read file: %w", err)
	}

	// 4. 检测真实 MIME 类型
	mtype := mimetype.Detect(data)
	mimeType := mtype.String()

	// 5. 检查文件类型是否允许
	if err := s.checkFileType(mimeType, params.Filename); err != nil {
		return nil, err
	}

	// 6. 判断文件类型分类
	fileType := classifyFileType(mimeType)

	// 7. 计算 MD5
	hash := md5.Sum(data)
	hashMD5 := hex.EncodeToString(hash[:])

	// 8. 去重检查
	if !s.cfg.AllowDuplicate {
		existing, err := s.fileRepo.GetByHashMD5(ctx, params.UserID, hashMD5)
		if err == nil && existing != nil {
			return &UploadResult{
				File:  existing,
				Links: s.generateLinks(existing, params.BaseURL),
			}, nil
		}
	}

	// 9. 获取默认存储驱动
	driver, storageConfigID, err := s.storageMgr.Default()
	if err != nil {
		return nil, fmt.Errorf("upload get storage: %w", err)
	}

	// 10. 生成存储 key
	ext := strings.TrimPrefix(filepath.Ext(params.Filename), ".")
	if ext == "" {
		ext = mtype.Extension()
		ext = strings.TrimPrefix(ext, ".")
	}
	fileID := uuid.New().String()
	storageKey := s.generateStorageKey(hashMD5, fileID, ext)

	// 11. 上传文件到存储后端
	if err := driver.Put(ctx, storageKey, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
		return nil, fmt.Errorf("upload put file: %w", err)
	}

	// 12. 如果是图片，获取尺寸并生成缩略图
	var width, height *int
	var thumbURL *string

	if fileType == model.FileTypeImage {
		dims, err := s.imageSvc.GetDimensions(bytes.NewReader(data))
		if err == nil {
			width = &dims.Width
			height = &dims.Height
		}

		thumbBuf, err := s.imageSvc.GenerateThumbnail(bytes.NewReader(data))
		if err == nil {
			thumbKey := "thumb/" + storageKey
			if putErr := driver.Put(ctx, thumbKey, thumbBuf, int64(thumbBuf.Len()), "image/jpeg"); putErr == nil {
				u := "/t/" + hashMD5[:8]
				thumbURL = &u
			}
		}
	}

	// 13. 生成访问 URL
	accessURL := s.buildAccessURL(fileType, hashMD5[:8])

	// 14. 创建文件记录
	file := &model.File{
		ID:              fileID,
		UserID:          params.UserID,
		AlbumID:         params.AlbumID,
		StorageConfigID: storageConfigID,
		OriginalName:    params.Filename,
		StorageKey:      storageKey,
		HashMD5:         hashMD5,
		SizeBytes:       int64(len(data)),
		MimeType:        mimeType,
		FileType:        fileType,
		Width:           width,
		Height:          height,
		URL:             accessURL,
		ThumbURL:        thumbURL,
	}

	if err := s.fileRepo.Create(ctx, file); err != nil {
		// 回滚：删除已上传的文件
		_ = driver.Delete(ctx, storageKey)
		return nil, fmt.Errorf("upload create record: %w", err)
	}

	// 15. 更新用户已用存储量（游客模式跳过）
	if !params.IsGuest {
		if err := s.userRepo.UpdateStorageUsed(ctx, params.UserID, int64(len(data))); err != nil {
			// 非致命错误，记录日志即可
			_ = err
		}
	}

	return &UploadResult{
		File:  file,
		Links: s.generateLinks(file, params.BaseURL),
	}, nil
}

// GetFile 获取文件详情。
func (s *FileService) GetFile(ctx context.Context, id string) (*model.File, error) {
	file, err := s.fileRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrFileNotFound
	}
	return file, nil
}

// DeleteFile 删除文件（软删除）。
func (s *FileService) DeleteFile(ctx context.Context, fileID, userID, role string) error {
	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return ErrFileNotFound
	}

	// 非管理员只能删除自己的文件
	if role != "admin" && file.UserID != userID {
		return ErrNotFileOwner
	}

	if err := s.fileRepo.SoftDelete(ctx, fileID); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	// 更新用户已用存储量
	_ = s.userRepo.UpdateStorageUsed(ctx, file.UserID, -file.SizeBytes)

	return nil
}

// ListFiles 查询文件列表。
func (s *FileService) ListFiles(ctx context.Context, params repo.FileListParams) ([]model.File, int64, error) {
	return s.fileRepo.List(ctx, params)
}

// GetSourceURL 返回文件在存储后端中的源站 URL。
func (s *FileService) GetSourceURL(file *model.File, baseURL string) string {
	driver, err := s.storageMgr.Get(file.StorageConfigID)
	if err != nil {
		return ""
	}
	sourceURL := driver.URL(file.StorageKey)
	if sourceURL == "" {
		return ""
	}
	base := strings.TrimRight(baseURL, "/")
	if base != "" && !strings.HasPrefix(sourceURL, "http") {
		sourceURL = base + sourceURL
	}
	return sourceURL
}

// GetFileContent 获取文件内容流（用于文件访问/下载）。
func (s *FileService) GetFileContent(ctx context.Context, file *model.File) (io.ReadCloser, int64, error) {
	driver, err := s.storageMgr.Get(file.StorageConfigID)
	if err != nil {
		return nil, 0, fmt.Errorf("get storage driver: %w", err)
	}
	return driver.Get(ctx, file.StorageKey)
}

// GetThumbContent 获取缩略图内容流（用于 /t/:hash 短链服务）。
// 缩略图的存储 key 固定为 "thumb/" + file.StorageKey。
func (s *FileService) GetThumbContent(ctx context.Context, file *model.File) (io.ReadCloser, int64, error) {
	driver, err := s.storageMgr.Get(file.StorageConfigID)
	if err != nil {
		return nil, 0, fmt.Errorf("get storage driver: %w", err)
	}
	return driver.Get(ctx, "thumb/"+file.StorageKey)
}

// GetFileByHash 通过 MD5 哈希前缀查找文件（用于公开访问链接）。
func (s *FileService) GetFileByHash(ctx context.Context, hashPrefix string) (*model.File, error) {
	// 通过 hash 前缀查询
	var file model.File
	// 这里需要在 repo 层额外添加方法，暂时直接用 Like 查询
	return &file, fmt.Errorf("get file by hash: not implemented via service, use repo directly")
}

func (s *FileService) checkFileType(mimeType, filename string) error {
	// 检查禁止的扩展名
	ext := strings.ToLower(filepath.Ext(filename))
	for _, forbidden := range s.cfg.ForbiddenExts {
		if ext == forbidden {
			return ErrFileTypeDenied
		}
	}

	// 检查允许的 MIME 类型
	if len(s.cfg.AllowedTypes) > 0 {
		allowed := false
		for _, prefix := range s.cfg.AllowedTypes {
			if strings.HasPrefix(mimeType, prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return ErrFileTypeDenied
		}
	}

	return nil
}

func (s *FileService) generateStorageKey(hashMD5, fileID, ext string) string {
	now := time.Now()
	return fmt.Sprintf("%d/%02d/%s/%s.%s",
		now.Year(), now.Month(), hashMD5[:8], fileID, ext)
}

func (s *FileService) buildAccessURL(fileType, hashShort string) string {
	var prefix string
	switch fileType {
	case model.FileTypeImage:
		prefix = "/i/"
	case model.FileTypeVideo:
		prefix = "/v/"
	case model.FileTypeAudio:
		prefix = "/a/"
	default:
		prefix = "/f/"
	}
	return prefix + hashShort
}

func (s *FileService) generateLinks(file *model.File, baseURL string) FileLinks {
	base := strings.TrimRight(baseURL, "/")
	url := file.URL
	if base != "" && !strings.HasPrefix(url, "http") {
		url = base + url
	}

	links := FileLinks{
		URL:              url,
		HTML:             fmt.Sprintf(`<img src="%s">`, url),
		BBCode:           fmt.Sprintf(`[img]%s[/img]`, url),
		Markdown:         fmt.Sprintf(`![%s](%s)`, file.OriginalName, url),
		MarkdownWithLink: fmt.Sprintf(`[![%s](%s)](%s)`, file.OriginalName, url, url),
	}
	if driver, err := s.storageMgr.Get(file.StorageConfigID); err == nil {
		if sourceURL := driver.URL(file.StorageKey); sourceURL != "" {
			if base != "" && !strings.HasPrefix(sourceURL, "http") {
				sourceURL = base + sourceURL
			}
			links.SourceURL = sourceURL
		}
	}
	if file.ThumbURL != nil {
		thumbURL := *file.ThumbURL
		if base != "" && !strings.HasPrefix(thumbURL, "http") {
			thumbURL = base + thumbURL
		}
		links.ThumbnailURL = thumbURL
	}
	return links
}

// classifyFileType 根据 MIME 类型判断文件分类。
func classifyFileType(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return model.FileTypeImage
	case strings.HasPrefix(mimeType, "video/"):
		return model.FileTypeVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return model.FileTypeAudio
	default:
		return model.FileTypeFile
	}
}
