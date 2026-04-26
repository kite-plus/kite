package router

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/i18n"
	"github.com/kite-plus/kite/internal/middleware"
	"github.com/kite-plus/kite/internal/model"
	"github.com/kite-plus/kite/internal/repo"
	"github.com/kite-plus/kite/internal/service"
)

// publicUser is the subset of user fields the landing-page templates need to
// render a login-aware header. It is populated from the access_token cookie
// when present and is always safe to render as nil.
type publicUser struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   string
	Initial     string
	Role        string
	IsAdmin     bool
}

// registerLanding wires the server-rendered landing pages (/, /explore,
// /upload, /share/:hash) and parses the embedded template set. These pages
// recognise the access_token cookie so the header reflects login state, but
// they never require authentication.
func registerLanding(r *gin.Engine, cfg Config, userRepo *repo.UserRepo, fileRepo *repo.FileRepo, settingRepo *repo.SettingRepo, settingDefaults map[string]string) {
	if cfg.TemplateFS != nil {
		if tmpl, err := template.ParseFS(cfg.TemplateFS, "layouts/*.html", "pages/*.html"); err == nil {
			r.SetHTMLTemplate(tmpl)
		}
	}

	r.GET("/", func(c *gin.Context) {
		settings := loadResolvedSettings(c.Request.Context(), settingRepo, settingDefaults)
		data := landingTemplateData(c, getOptionalUser(c, cfg.AuthSvc, userRepo), settings, "", "")
		c.HTML(http.StatusOK, "index.html", data)
	})

	r.GET("/explore", func(c *gin.Context) {
		settings := loadResolvedSettings(c.Request.Context(), settingRepo, settingDefaults)
		locale := middleware.LocaleFromGin(c)
		data := landingTemplateData(c, getOptionalUser(c, cfg.AuthSvc, userRepo), settings, "explore", i18n.T(locale, "explore.title"))
		data["GalleryEnabled"] = strings.EqualFold(settings[service.AllowPublicGallerySettingKey], "true")
		c.HTML(http.StatusOK, "explore.html", data)
	})

	r.GET("/upload", func(c *gin.Context) {
		settings := loadResolvedSettings(c.Request.Context(), settingRepo, settingDefaults)
		locale := middleware.LocaleFromGin(c)
		data := landingTemplateData(c, getOptionalUser(c, cfg.AuthSvc, userRepo), settings, "upload", i18n.T(locale, "upload.title"))
		data["GuestUploadEnabled"] = strings.EqualFold(settings[service.AllowGuestUploadSettingKey], "true")
		uploadMaxFileSizeBytes := resolveUploadMaxFileSizeBytes(settings, cfg.UploadMaxFileSize)
		data["UploadMaxFileSizeBytes"] = uploadMaxFileSizeBytes
		data["UploadMaxFileSizeLabel"] = formatUploadMaxFileSizeLabel(uploadMaxFileSizeBytes)
		c.HTML(http.StatusOK, "upload.html", data)
	})

	r.GET("/share/:hash", func(c *gin.Context) {
		settings := loadResolvedSettings(c.Request.Context(), settingRepo, settingDefaults)
		user := getOptionalUser(c, cfg.AuthSvc, userRepo)
		locale := middleware.LocaleFromGin(c)

		hash := c.Param("hash")
		file, err := fileRepo.GetByHashPrefix(c.Request.Context(), hash)
		if err != nil {
			data := landingTemplateData(c, user, settings, "", i18n.T(locale, "share.not_found_title"))
			data["NotFound"] = true
			c.HTML(http.StatusNotFound, "share.html", data)
			return
		}

		baseURL := requestBaseURL(c)
		var sourceURL string
		if cfg.FileSvc != nil {
			sourceURL = cfg.FileSvc.GetSourceURL(c.Request.Context(), file, baseURL)
		}

		fileView := buildShareFileView(c, file, baseURL, sourceURL)

		// Server-side syntax highlighting: when the file is text-shaped, read
		// up to shareTextPreviewCap bytes and run them through chroma so the
		// template gets a fully-rendered <pre><code>...</code></pre> block.
		// Avoids the previous client-side fetch + flash-of-unstyled-content.
		if cfg.FileSvc != nil && fileView["IsText"] == true {
			if reader, _, fcErr := cfg.FileSvc.GetFileContent(c.Request.Context(), file); fcErr == nil {
				html, truncated, hlErr := highlightShareText(reader, file.OriginalName, file.MimeType)
				reader.Close()
				if hlErr == nil {
					fileView["HighlightedHTML"] = template.HTML(html)
					fileView["HighlightTruncated"] = truncated
				}
			}
		}

		data := landingTemplateData(c, user, settings, "", file.OriginalName)
		data["File"] = fileView
		data["HighlightCSS"] = template.CSS(shareHighlightCSS)
		c.HTML(http.StatusOK, "share.html", data)
	})
}

// requestBaseURL mirrors handler.RequestBaseURL — duplicated here to avoid the
// router→handler import cycle that would arise from sharing the helper.
func requestBaseURL(c *gin.Context) string {
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if c.Request.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + c.Request.Host
}

// buildShareFileView assembles the per-file fields the share.html template
// reads. Raw URLs are picked per file_type so the template can drop the URL
// directly into the right element without re-deriving it. Sizes/dates are
// pre-formatted server-side to keep the template free of helpers.
//
// The gin context is required so the "Source URL" link label can be
// translated alongside the rest of the page — the other labels (URL,
// Markdown, HTML, BBCode) are technical identifiers so they stay in
// English regardless of locale.
func buildShareFileView(c *gin.Context, file *model.File, baseURL, sourceURL string) gin.H {
	var rawPath, thumbPath string
	switch file.FileType {
	case model.FileTypeImage:
		rawPath = "/i/" + file.HashMD5
		thumbPath = "/t/" + file.HashMD5
	case model.FileTypeVideo:
		rawPath = "/v/" + file.HashMD5
	case model.FileTypeAudio:
		rawPath = "/a/" + file.HashMD5
	default:
		rawPath = "/f/" + file.HashMD5
	}

	ext := ""
	if dot := strings.LastIndex(file.OriginalName, "."); dot >= 0 && dot < len(file.OriginalName)-1 {
		ext = strings.ToLower(file.OriginalName[dot+1:])
	}

	isPDF := strings.EqualFold(file.MimeType, "application/pdf") || ext == "pdf"
	previewPath := rawPath
	if isPDF {
		// /f/<hash> defaults to attachment disposition. Opt into inline so
		// the iframe in share.html can render the PDF instead of triggering
		// a download.
		previewPath = rawPath + "?inline=1"
	}

	absoluteURL := baseURL + rawPath

	locale := middleware.LocaleFromGin(c)
	links := []gin.H{
		{"Label": "URL", "Value": absoluteURL},
	}
	if sourceURL != "" {
		links = append(links, gin.H{"Label": i18n.T(locale, "share.source_url_label"), "Value": sourceURL})
	}
	links = append(links,
		gin.H{"Label": "Markdown", "Value": "![" + file.OriginalName + "](" + absoluteURL + ")"},
		gin.H{"Label": "HTML", "Value": `<img src="` + absoluteURL + `" alt="` + file.OriginalName + `">`},
		gin.H{"Label": "BBCode", "Value": "[img]" + absoluteURL + "[/img]"},
	)

	return gin.H{
		"Hash":         file.HashMD5,
		"OriginalName": file.OriginalName,
		"FileType":     file.FileType,
		"MimeType":     file.MimeType,
		"SizeLabel":    formatShareSize(file.SizeBytes),
		"Width":        file.Width,
		"Height":       file.Height,
		"Duration":     file.Duration,
		"DurationText": formatShareDuration(file.Duration),
		"CreatedAt":    file.CreatedAt.Format("2006-01-02 15:04"),
		"RawURL":       rawPath,
		"PreviewURL":   previewPath,
		"DownloadURL":  "/f/" + file.HashMD5 + "?dl=1",
		"ThumbURL":     thumbPath,
		"AbsoluteURL":  absoluteURL,
		"Links":        links,
		"Ext":          ext,
		"IsImage":      file.FileType == model.FileTypeImage,
		"IsVideo":      file.FileType == model.FileTypeVideo,
		"IsAudio":      file.FileType == model.FileTypeAudio,
		"IsPDF":        isPDF,
		"IsText":       isShareTextLike(file.MimeType, ext),
	}
}

// isShareTextLike mirrors the previous React component's text-detection rules
// so the same MIME types and extensions get an inline preview.
func isShareTextLike(mime, ext string) bool {
	m := strings.ToLower(mime)
	prefixes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-yaml",
		"application/x-sh",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(m, p) {
			return true
		}
	}
	textExts := map[string]bool{
		"md": true, "markdown": true, "txt": true, "log": true, "csv": true, "tsv": true,
		"json": true, "yaml": true, "yml": true, "xml": true, "html": true, "htm": true,
		"css": true, "js": true, "ts": true, "tsx": true, "jsx": true, "go": true, "py": true,
		"rs": true, "java": true, "c": true, "h": true, "cpp": true, "rb": true, "sh": true,
		"sql": true, "toml": true, "ini": true, "conf": true,
	}
	return textExts[ext]
}

func formatShareSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatShareDuration(d *int) string {
	if d == nil || *d <= 0 {
		return ""
	}
	s := *d
	h := s / 3600
	m := (s % 3600) / 60
	sec := s % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%d:%02d", m, sec)
}

// getOptionalUser decodes the access_token cookie (if any) into a publicUser
// view used by landing templates. Returns nil for anonymous visitors or when
// the token is invalid or expired. Never returns an error since every failure
// mode degrades to "not logged in".
func getOptionalUser(c *gin.Context, authSvc *service.AuthService, userRepo *repo.UserRepo) *publicUser {
	cookie, err := c.Cookie("access_token")
	if err != nil || cookie == "" {
		return nil
	}
	claims, err := authSvc.ValidateToken(cookie)
	if err != nil {
		return nil
	}

	view := &publicUser{
		ID:          claims.UserID,
		Username:    claims.Username,
		DisplayName: claims.Username,
		Role:        claims.Role,
		IsAdmin:     claims.Role == "admin",
	}

	if user, getErr := userRepo.GetByID(c.Request.Context(), claims.UserID); getErr == nil {
		view.Username = user.Username
		if user.Nickname != nil {
			if nickname := strings.TrimSpace(*user.Nickname); nickname != "" {
				view.DisplayName = nickname
			}
		}
		if user.AvatarURL != nil {
			view.AvatarURL = strings.TrimSpace(*user.AvatarURL)
		}
	}

	view.Initial = nameInitial(view.Username)
	return view
}

// nameInitial returns the uppercase first rune of name, or "U" when name is
// empty or decoding fails. Used as the avatar fallback glyph.
func nameInitial(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "U"
	}
	r, _ := utf8.DecodeRuneInString(trimmed)
	if r == utf8.RuneError {
		return "U"
	}
	return strings.ToUpper(string(r))
}
