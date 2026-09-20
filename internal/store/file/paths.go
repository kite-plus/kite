package file

import (
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/kite-plus/kite/internal/content"
)

// ContentDir is the project subdirectory holding all content.
const ContentDir = "content"

// BundleIndex is the file that carries an item's text inside a bundle.
const BundleIndex = "index.md"

// markdownExt is the only source extension recognized today.
const markdownExt = ".md"

// Slugify turns a title into a URL fragment. Letters and digits of any script
// are kept, so a CJK title yields a readable slug rather than an empty one.
func Slugify(s string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.TrimSpace(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// LocatorFor returns where a new item's bytes belong.
//
// The bundle directory name is derived from the slug once, at creation time,
// and then never changes: slug, path and URL are independent, so renaming a
// slug must not move files (see docs/design/architecture.md D2).
func LocatorFor(t *content.Type, key string) content.Locator {
	dir := path.Join(ContentDir, t.Dir)
	if t.Layout == content.LayoutBundle {
		return content.Locator(path.Join(dir, key))
	}
	return content.Locator(path.Join(dir, key+markdownExt))
}

// SourcePath returns the markdown file inside a locator, relative to the
// project root and slash separated.
func SourcePath(t *content.Type, loc content.Locator) string {
	if t.Layout == content.LayoutBundle {
		return path.Join(string(loc), BundleIndex)
	}
	return string(loc)
}

// MediaDir returns the directory an item's media belongs in, or "" when the
// layout has no bundle to hold it.
func MediaDir(t *content.Type, loc content.Locator) string {
	if t.Layout == content.LayoutBundle {
		return string(loc)
	}
	return ""
}

// abs joins a slash separated project-relative path onto the root.
func abs(root, rel string) string {
	return filepath.Join(root, filepath.FromSlash(rel))
}

// rel converts an absolute path back into a slash separated project-relative
// path.
func rel(root, absPath string) (string, error) {
	r, err := filepath.Rel(root, absPath)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(r), nil
}

// IsMarkdown reports whether a filename is a content source file.
func IsMarkdown(name string) bool {
	return strings.EqualFold(filepath.Ext(name), markdownExt)
}
