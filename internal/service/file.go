package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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

// FileService implements the core file-handling business logic.
type FileService struct {
	fileRepo    *repo.FileRepo
	userRepo    *repo.UserRepo
	storageRepo *repo.StorageConfigRepo
	replicaRepo *repo.FileReplicaRepo
	settingRepo *repo.SettingRepo
	storageMgr  *storage.Manager
	router      *storage.Router
	imageSvc    *ImageService
	cfg         config.UploadConfig
}

func NewFileService(
	fileRepo *repo.FileRepo,
	userRepo *repo.UserRepo,
	storageRepo *repo.StorageConfigRepo,
	replicaRepo *repo.FileReplicaRepo,
	settingRepo *repo.SettingRepo,
	storageMgr *storage.Manager,
	router *storage.Router,
	imageSvc *ImageService,
	cfg config.UploadConfig,
) *FileService {
	return &FileService{
		fileRepo:    fileRepo,
		userRepo:    userRepo,
		storageRepo: storageRepo,
		replicaRepo: replicaRepo,
		settingRepo: settingRepo,
		storageMgr:  storageMgr,
		router:      router,
		imageSvc:    imageSvc,
		cfg:         cfg,
	}
}

// UploadParams carries the parameters for an upload.
type UploadParams struct {
	UserID   string
	AlbumID  *string
	Filename string
	Reader   io.Reader
	Size     int64
	IsGuest  bool   // when true, skip per-user quota checks (anonymous uploads)
	BaseURL  string // request origin (e.g. "https://www.kite.plus") used to build absolute links
}

// UploadResult is the outcome of an upload.
type UploadResult struct {
	File  *model.File
	Links FileLinks
}

// FileLinks bundles access links in assorted formats, matching the Lsky schema.
type FileLinks struct {
	URL              string `json:"url"`                  // short link, e.g. /i/{hash}
	SourceURL        string `json:"source_url,omitempty"` // raw storage URL pointing at the backing path
	HTML             string `json:"html"`
	BBCode           string `json:"bbcode"`
	Markdown         string `json:"markdown"`
	MarkdownWithLink string `json:"markdown_with_link"`
	ThumbnailURL     string `json:"thumbnail_url,omitempty"`
}

// Upload runs the full upload pipeline.
func (s *FileService) Upload(ctx context.Context, params UploadParams) (*UploadResult, error) {
	// 1. Enforce the per-file size limit.
	maxFileSize := s.maxFileSize(ctx)
	if params.Size > maxFileSize {
		return nil, ErrFileTooLarge
	}

	// 2. Optimistic preflight against declared size — rejects users already over
	// quota before we bother reading the body. NOT authoritative (the declared
	// size can lie, and concurrent uploads race), so step 9 re-checks atomically
	// against the real byte count.
	if !params.IsGuest {
		user, err := s.userRepo.GetByID(ctx, params.UserID)
		if err != nil {
			return nil, fmt.Errorf("upload get user: %w", err)
		}
		if !user.HasStorageSpace(params.Size) {
			return nil, ErrStorageFull
		}
	}

	// 3. Read the entire body into memory for MIME detection and MD5 hashing.
	data, err := io.ReadAll(params.Reader)
	if err != nil {
		return nil, fmt.Errorf("upload read file: %w", err)
	}
	if int64(len(data)) > maxFileSize {
		return nil, ErrFileTooLarge
	}

	// 4. Detect the real MIME type.
	mtype := mimetype.Detect(data)
	detectedMimeType := mtype.String()

	// 5. Resolve dangerous-extension policy before dedupe so blocked files are
	// rejected immediately and renamed files do not accidentally reuse unsafe
	// historical records.
	effectiveName, mimeType, fileType, renameSuffix, isDangerousRenamed, err := s.resolveUploadIdentity(ctx, params.Filename, detectedMimeType)
	if err != nil {
		return nil, err
	}

	// 6. Enforce the allow-list of MIME types, if any.
	if err := s.checkAllowedMimeType(mimeType); err != nil {
		return nil, err
	}

	// 7. Compute the MD5 hash.
	hash := md5.Sum(data)
	hashMD5 := hex.EncodeToString(hash[:])

	// 8. Deduplicate against existing uploads.
	if !s.cfg.AllowDuplicate {
		existing, err := s.findDuplicateFile(ctx, params.UserID, hashMD5, isDangerousRenamed, renameSuffix)
		if err == nil && existing != nil {
			return &UploadResult{
				File:  existing,
				Links: s.generateLinks(ctx, existing, params.BaseURL),
			}, nil
		}
	}

	// 9. Atomically reserve quota against the REAL byte count. This is the
	// authoritative gate — the check and the counter increment happen inside a
	// single conditional UPDATE, so concurrent uploads cannot race past the
	// limit. Any subsequent failure must release the reservation to keep the
	// counter honest. Guests (anonymous uploads) bypass per-user quota entirely.
	uploadSize := int64(len(data))
	quotaReserved := false
	if !params.IsGuest {
		ok, err := s.userRepo.TryConsumeStorage(ctx, params.UserID, uploadSize)
		if err != nil {
			return nil, fmt.Errorf("upload reserve quota: %w", err)
		}
		if !ok {
			return nil, ErrStorageFull
		}
		quotaReserved = true
	}

	// releaseQuota undoes the reservation. Deferred so that any early return
	// after quota was reserved unwinds the counter; on success we clear the
	// flag so the defer becomes a no-op.
	defer func() {
		if quotaReserved {
			if relErr := s.userRepo.ReleaseStorage(ctx, params.UserID, uploadSize); relErr != nil {
				log.Printf("upload: release quota after failure for user=%s size=%d: %v", params.UserID, uploadSize, relErr)
			}
		}
	}()

	// 10. Pick upload targets according to the policy (single / primary_fallback / round_robin / mirror).
	plan, err := s.router.Plan(ctx, uploadSize)
	if err != nil {
		return nil, ErrStorageFull
	}

	// 11. Build the storage key.
	ext := strings.TrimPrefix(filepath.Ext(effectiveName), ".")
	if ext == "" {
		ext = mtype.Extension()
		ext = strings.TrimPrefix(ext, ".")
	}
	fileID := uuid.New().String()
	storageKey, err := s.generateStorageKey(ctx, UploadPathPatternData{
		Now:      time.Now(),
		UserID:   params.UserID,
		FileType: fileType,
		HashMD5:  hashMD5,
		FileID:   fileID,
		Ext:      ext,
	})
	if err != nil {
		return nil, fmt.Errorf("upload generate storage key: %w", err)
	}

	// 12. Write to the primary storage according to the policy.
	primary, replicaTargets, err := s.writePrimary(ctx, plan, storageKey, data, mimeType)
	if err != nil {
		return nil, fmt.Errorf("upload put file: %w", err)
	}

	// 13. For images, capture dimensions and generate a thumbnail on the primary storage only.
	var width, height *int
	var thumbURL *string

	if fileType == model.FileTypeImage {
		dims, err := s.imageSvc.GetDimensions(bytes.NewReader(data))
		if err == nil {
			width = &dims.Width
			height = &dims.Height
		}

		thumbBuf, thumbMime, err := s.imageSvc.GenerateThumbnail(bytes.NewReader(data), mimeType)
		if err == nil {
			thumbKey := "thumb/" + storageKey
			if putErr := primary.Driver.Put(ctx, thumbKey, thumbBuf, int64(thumbBuf.Len()), thumbMime); putErr == nil {
				u := "/t/" + hashMD5[:8]
				thumbURL = &u
			}
		}
	}

	// 14. Build the access URL.
	accessURL := s.buildAccessURL(fileType, hashMD5[:8])

	// 15. Create the file record.
	file := &model.File{
		ID:              fileID,
		UserID:          params.UserID,
		AlbumID:         params.AlbumID,
		StorageConfigID: primary.Meta.ID,
		OriginalName:    effectiveName,
		StorageKey:      storageKey,
		HashMD5:         hashMD5,
		SizeBytes:       uploadSize,
		MimeType:        mimeType,
		FileType:        fileType,
		Width:           width,
		Height:          height,
		URL:             accessURL,
		ThumbURL:        thumbURL,
	}

	if err := s.fileRepo.Create(ctx, file); err != nil {
		// Roll back by deleting the uploaded bytes. Quota is released by the
		// deferred releaseQuota above because quotaReserved is still true.
		_ = primary.Driver.Delete(ctx, storageKey)
		return nil, fmt.Errorf("upload create record: %w", err)
	}

	// 16. In mirror mode, pre-insert replica records and kick off concurrent background replication.
	if plan.Mode == storage.PolicyMirror && len(replicaTargets) > 0 {
		s.scheduleReplicas(file, data, mimeType, replicaTargets)
	}

	// Upload succeeded — the quota reservation in step 9 is now permanent.
	// Clearing the flag disarms the deferred release so the counter stays bumped.
	quotaReserved = false

	return &UploadResult{
		File:  file,
		Links: s.generateLinks(ctx, file, params.BaseURL),
	}, nil
}

// writePrimary writes to the primary storage per the policy and returns the successful target and any
// replica targets to copy asynchronously.
// primary_fallback: try targets in order until one succeeds; the rest are unused fallbacks and are not replicated.
// mirror: Targets[0] is the primary, Targets[1:] must be replicated asynchronously.
// single / round_robin: Targets[0] is the primary and no replicas are produced.
func (s *FileService) writePrimary(
	ctx context.Context,
	plan *storage.Plan,
	storageKey string,
	data []byte,
	mimeType string,
) (primary storage.Target, replicas []storage.Target, err error) {
	if len(plan.Targets) == 0 {
		return storage.Target{}, nil, fmt.Errorf("empty plan")
	}

	putOnce := func(t storage.Target) error {
		return t.Driver.Put(ctx, storageKey, bytes.NewReader(data), int64(len(data)), mimeType)
	}

	switch plan.Mode {
	case storage.PolicyPrimaryFallback:
		var lastErr error
		for _, t := range plan.Targets {
			if putErr := putOnce(t); putErr == nil {
				return t, nil, nil
			} else {
				lastErr = putErr
				log.Printf("storage fallback: %s (%s) failed, trying next: %v", t.Meta.Name, t.Meta.ID, putErr)
			}
		}
		return storage.Target{}, nil, fmt.Errorf("all fallback storages failed: %w", lastErr)

	case storage.PolicyMirror:
		primary = plan.Targets[0]
		if putErr := putOnce(primary); putErr != nil {
			return storage.Target{}, nil, fmt.Errorf("mirror primary %q: %w", primary.Meta.Name, putErr)
		}
		return primary, plan.Targets[1:], nil

	default: // single / round_robin
		primary = plan.Targets[0]
		if putErr := putOnce(primary); putErr != nil {
			return storage.Target{}, nil, fmt.Errorf("put %q: %w", primary.Meta.Name, putErr)
		}
		return primary, nil, nil
	}
}

// scheduleReplicas pre-inserts a pending row for each mirror replica, then spawns concurrent background workers.
// Workers use an independent context (not derived from the request) and run one per replica; failures are
// recorded as status=failed with an error message and do not affect the main request.
func (s *FileService) scheduleReplicas(file *model.File, data []byte, mimeType string, targets []storage.Target) {
	ctx := context.Background()

	type job struct {
		replicaID string
		target    storage.Target
	}
	jobs := make([]job, 0, len(targets))
	for _, t := range targets {
		r := &model.FileReplica{
			ID:              uuid.New().String(),
			FileID:          file.ID,
			StorageConfigID: t.Meta.ID,
			Status:          model.ReplicaStatusPending,
		}
		if err := s.replicaRepo.Create(ctx, r); err != nil {
			log.Printf("mirror: create replica record failed for %s: %v", t.Meta.Name, err)
			continue
		}
		jobs = append(jobs, job{replicaID: r.ID, target: t})
	}

	for _, j := range jobs {
		j := j
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			err := j.target.Driver.Put(bgCtx, file.StorageKey, bytes.NewReader(data), int64(len(data)), mimeType)
			if err != nil {
				log.Printf("mirror replicate %s (%s) failed: %v", j.target.Meta.Name, j.target.Meta.ID, err)
				_ = s.replicaRepo.UpdateStatus(bgCtx, j.replicaID, model.ReplicaStatusFailed, err.Error())
				return
			}
			_ = s.replicaRepo.UpdateStatus(bgCtx, j.replicaID, model.ReplicaStatusOK, "")
		}()
	}
}

// GetFile fetches a file by ID.
func (s *FileService) GetFile(ctx context.Context, id string) (*model.File, error) {
	file, err := s.fileRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrFileNotFound
	}
	return file, nil
}

// DeleteFile soft-deletes a file.
func (s *FileService) DeleteFile(ctx context.Context, fileID, userID, role string) error {
	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return ErrFileNotFound
	}

	// Non-admins may only delete their own files.
	if role != "admin" && file.UserID != userID {
		return ErrNotFileOwner
	}

	if err := s.fileRepo.SoftDelete(ctx, fileID); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	// Update the user's storage usage.
	_ = s.userRepo.UpdateStorageUsed(ctx, file.UserID, -file.SizeBytes)

	return nil
}

// ListFiles queries the file list.
func (s *FileService) ListFiles(ctx context.Context, params repo.FileListParams) ([]model.File, int64, error) {
	return s.fileRepo.List(ctx, params)
}

// GetSourceURL returns the file's origin URL on its storage backend.
func (s *FileService) GetSourceURL(ctx context.Context, file *model.File, baseURL string) string {
	driver, err := s.storageDriverForConfig(ctx, file.StorageConfigID)
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

// GetFileContent opens the file content stream, used for access and download.
func (s *FileService) GetFileContent(ctx context.Context, file *model.File) (io.ReadCloser, int64, error) {
	driver, err := s.storageDriverForConfig(ctx, file.StorageConfigID)
	if err != nil {
		return nil, 0, fmt.Errorf("get storage driver: %w", err)
	}
	return driver.Get(ctx, file.StorageKey)
}

// GetThumbContent opens the thumbnail stream for the /t/:hash short-link endpoint.
// The thumbnail storage key is always "thumb/" + file.StorageKey.
func (s *FileService) GetThumbContent(ctx context.Context, file *model.File) (io.ReadCloser, int64, error) {
	driver, err := s.storageDriverForConfig(ctx, file.StorageConfigID)
	if err != nil {
		return nil, 0, fmt.Errorf("get storage driver: %w", err)
	}
	return driver.Get(ctx, "thumb/"+file.StorageKey)
}

// RegenerateThumbnail rebuilds the thumbnail for an image file from its source bytes and
// persists it back to the same storage. Used as a self-healing fallback when GetThumbContent
// fails — the thumb file may have been deleted out-of-band, or the original upload may have
// left an orphaned thumb_url without actually writing the thumb.
// Returns a reader positioned at the start of the freshly encoded thumbnail.
func (s *FileService) RegenerateThumbnail(ctx context.Context, file *model.File) (io.ReadCloser, int64, error) {
	if file.FileType != model.FileTypeImage {
		return nil, 0, fmt.Errorf("regenerate thumbnail: not an image")
	}

	driver, err := s.storageDriverForConfig(ctx, file.StorageConfigID)
	if err != nil {
		return nil, 0, fmt.Errorf("regenerate thumbnail: get storage driver: %w", err)
	}

	srcReader, _, err := driver.Get(ctx, file.StorageKey)
	if err != nil {
		return nil, 0, fmt.Errorf("regenerate thumbnail: read source: %w", err)
	}
	defer srcReader.Close()

	srcBytes, err := io.ReadAll(srcReader)
	if err != nil {
		return nil, 0, fmt.Errorf("regenerate thumbnail: load source: %w", err)
	}

	thumbBuf, thumbMime, err := s.imageSvc.GenerateThumbnail(bytes.NewReader(srcBytes), file.MimeType)
	if err != nil {
		return nil, 0, fmt.Errorf("regenerate thumbnail: encode: %w", err)
	}

	thumbBytes := thumbBuf.Bytes()
	thumbKey := "thumb/" + file.StorageKey
	if err := driver.Put(ctx, thumbKey, bytes.NewReader(thumbBytes), int64(len(thumbBytes)), thumbMime); err != nil {
		return nil, 0, fmt.Errorf("regenerate thumbnail: write: %w", err)
	}

	if file.ThumbURL == nil {
		url := "/t/" + file.HashMD5[:8]
		if err := s.fileRepo.UpdateThumbURL(ctx, file.ID, &url); err == nil {
			file.ThumbURL = &url
		}
	}

	return io.NopCloser(bytes.NewReader(thumbBytes)), int64(len(thumbBytes)), nil
}

// GetFileByHash resolves a file by its MD5 hash prefix, used for public access links.
func (s *FileService) GetFileByHash(ctx context.Context, hashPrefix string) (*model.File, error) {
	file, err := s.fileRepo.GetByHashPrefix(ctx, hashPrefix)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "record not found") ||
			strings.Contains(strings.ToLower(err.Error()), "prefix too short") {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("get file by hash: %w", err)
	}
	return file, nil
}

func (s *FileService) checkAllowedMimeType(mimeType string) error {
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

func (s *FileService) resolveUploadIdentity(ctx context.Context, filename, detectedMimeType string) (effectiveName, mimeType, fileType, renameSuffix string, renamed bool, err error) {
	settings := s.dangerousExtensionSettings(ctx)
	decision, matched, err := DecideDangerousExtension(filename, settings.Rules, settings.RenameSuffix)
	if err != nil {
		return "", "", "", "", false, fmt.Errorf("upload resolve dangerous extension: %w", err)
	}
	if matched && decision.Action == DangerousExtensionActionBlock {
		return "", "", "", "", false, ErrFileTypeDenied
	}
	if matched && decision.Action == DangerousExtensionActionRename {
		return decision.SafeName, DangerousRenameMimeType, model.FileTypeFile, settings.RenameSuffix, true, nil
	}
	return filename, detectedMimeType, classifyFileType(detectedMimeType), settings.RenameSuffix, false, nil
}

func (s *FileService) dangerousExtensionSettings(ctx context.Context) dangerousExtensionSettings {
	defaultRules := DefaultDangerousExtensionRules(s.cfg.ForbiddenExts)
	defaultSuffix := DefaultDangerousRenameSuffixValue

	if s.settingRepo == nil {
		return dangerousExtensionSettings{
			Rules:        ResolveDangerousExtensionRules(defaultRules, s.cfg.ForbiddenExts),
			RenameSuffix: defaultSuffix,
		}
	}

	rulesRaw, err := s.settingRepo.GetOrDefault(ctx, UploadDangerousExtensionRulesSettingKey, defaultRules)
	if err != nil {
		rulesRaw = defaultRules
	}
	suffixRaw, err := s.settingRepo.GetOrDefault(ctx, UploadDangerousRenameSuffixSettingKey, defaultSuffix)
	if err != nil {
		suffixRaw = defaultSuffix
	}

	return dangerousExtensionSettings{
		Rules:        ResolveDangerousExtensionRules(rulesRaw, s.cfg.ForbiddenExts),
		RenameSuffix: ResolveDangerousRenameSuffix(suffixRaw),
	}
}

func (s *FileService) findDuplicateFile(ctx context.Context, userID, hashMD5 string, renamed bool, renameSuffix string) (*model.File, error) {
	if !renamed {
		return s.fileRepo.GetByHashMD5(ctx, userID, hashMD5)
	}

	files, err := s.fileRepo.ListByHashMD5(ctx, userID, hashMD5)
	if err != nil {
		return nil, err
	}

	for i := range files {
		if isSafeDangerousDuplicate(&files[i], renameSuffix) {
			return &files[i], nil
		}
	}

	return nil, nil
}

func isSafeDangerousDuplicate(file *model.File, renameSuffix string) bool {
	if file == nil {
		return false
	}

	safeSuffix := ResolveDangerousRenameSuffix(renameSuffix)
	if file.MimeType != DangerousRenameMimeType {
		return false
	}
	if file.FileType != model.FileTypeFile {
		return false
	}
	if file.ThumbURL != nil {
		return false
	}
	if !strings.HasPrefix(file.URL, "/f/") {
		return false
	}
	if strings.ToLower(strings.TrimPrefix(filepath.Ext(file.OriginalName), ".")) != safeSuffix {
		return false
	}
	if strings.ToLower(strings.TrimPrefix(filepath.Ext(file.StorageKey), ".")) != safeSuffix {
		return false
	}
	return true
}

type dangerousExtensionSettings struct {
	Rules        []DangerousExtensionRule
	RenameSuffix string
}

func (s *FileService) generateStorageKey(ctx context.Context, data UploadPathPatternData) (string, error) {
	pattern := s.cfg.PathPattern
	if s.settingRepo != nil {
		if saved, err := s.settingRepo.GetOrDefault(ctx, UploadPathPatternSettingKey, s.cfg.PathPattern); err == nil {
			pattern = saved
		}
	}

	normalized, err := NormalizeUploadPathPattern(pattern)
	if err != nil {
		fallback, fallbackErr := NormalizeUploadPathPattern(s.cfg.PathPattern)
		if fallbackErr != nil {
			return "", err
		}
		normalized = fallback
	}

	return RenderUploadPathPattern(normalized, data)
}

func (s *FileService) maxFileSize(ctx context.Context) int64 {
	if s.settingRepo == nil {
		return s.cfg.MaxFileSize
	}

	raw, err := s.settingRepo.GetOrDefault(ctx, UploadMaxFileSizeMBSettingKey, DefaultUploadMaxFileSizeMB(s.cfg.MaxFileSize))
	if err != nil {
		return s.cfg.MaxFileSize
	}

	maxBytes, err := ParseUploadMaxFileSizeBytes(raw)
	if err != nil {
		return s.cfg.MaxFileSize
	}
	return maxBytes
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

func (s *FileService) generateLinks(ctx context.Context, file *model.File, baseURL string) FileLinks {
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
	if driver, err := s.storageDriverForConfig(ctx, file.StorageConfigID); err == nil {
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

// storageDriverForConfig resolves the driver's runtime instance for a file's
// storage configuration. Active storages come from the manager cache; historical
// files may still point at disabled storages, so those are rebuilt from the
// persisted config on demand.
func (s *FileService) storageDriverForConfig(ctx context.Context, configID string) (storage.StorageDriver, error) {
	if driver, err := s.storageMgr.Get(configID); err == nil {
		return driver, nil
	}
	if s.storageRepo == nil {
		return nil, fmt.Errorf("storage config %q is not available", configID)
	}

	cfg, err := s.storageRepo.GetByID(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("get storage config %q: %w", configID, err)
	}

	driverName, _ := storage.CanonicalDriverAndProvider(cfg.Driver, cfg.Provider, cfg.Config)
	parsed, err := storage.ParseConfig(driverName, json.RawMessage(cfg.Config))
	if err != nil {
		return nil, fmt.Errorf("parse storage config %q: %w", configID, err)
	}

	driver, err := storage.NewDriver(parsed)
	if err != nil {
		return nil, fmt.Errorf("build storage driver %q: %w", configID, err)
	}
	return driver, nil
}

// classifyFileType derives the file category from its MIME type.
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
