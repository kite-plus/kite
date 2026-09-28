// Package hexo moves a Hexo site's content into a Kite project.
//
// The Hexo site is read and never written. Its posts, drafts and pages become
// items written through the project's store, the files beside a post become
// its bundle's, the rest of source/ becomes the site's static files, and every
// address Hexo published an item at becomes an alias, so links to the old site
// keep working.
package hexo

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/project"
	"github.com/kite-plus/kite/internal/store/file"
)

// Config is what a Hexo site's _config.yml says that Kite has a use for.
type Config struct {
	Title       string
	Description string
	Author      string
	Language    string
	Timezone    string

	// URL is the address the site is published at, with the path under it.
	URL string

	// Permalink is the pattern a post's address follows, such as
	// :year/:month/:day/:title/, and PermalinkDefaults the values of its
	// tokens that a post leaves out.
	Permalink         string
	PermalinkDefaults map[string]string

	// SourceDir is the folder the content lives in, source by default.
	SourceDir string
}

// ReadConfig reads _config.yml at the root of a Hexo site.
func ReadConfig(root string) (Config, error) {
	data, err := os.ReadFile(filepath.Join(root, "_config.yml"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, fmt.Errorf("hexo: %s has no _config.yml; is it a Hexo site?", root)
		}
		return Config{}, err
	}
	var raw struct {
		Title             string            `yaml:"title"`
		Subtitle          string            `yaml:"subtitle"`
		Description       string            `yaml:"description"`
		Author            string            `yaml:"author"`
		Language          any               `yaml:"language"`
		Timezone          string            `yaml:"timezone"`
		URL               string            `yaml:"url"`
		Root              string            `yaml:"root"`
		Permalink         string            `yaml:"permalink"`
		PermalinkDefaults map[string]string `yaml:"permalink_defaults"`
		SourceDir         string            `yaml:"source_dir"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("hexo: read _config.yml: %w", err)
	}

	cfg := Config{
		Title:             strings.TrimSpace(raw.Title),
		Description:       strings.TrimSpace(cmpOr(raw.Description, raw.Subtitle)),
		Author:            strings.TrimSpace(raw.Author),
		Timezone:          strings.TrimSpace(raw.Timezone),
		Permalink:         cmpOr(strings.TrimSpace(raw.Permalink), ":year/:month/:day/:title/"),
		PermalinkDefaults: raw.PermalinkDefaults,
		SourceDir:         cmpOr(strings.Trim(strings.TrimSpace(raw.SourceDir), "/"), "source"),
	}
	switch lang := raw.Language.(type) {
	case string:
		cfg.Language = strings.TrimSpace(lang)
	case []any:
		if len(lang) > 0 {
			cfg.Language = strings.TrimSpace(fmt.Sprint(lang[0]))
		}
	}
	// Hexo keeps the path a site is published under in root, and since
	// Hexo 5 in url as well.
	cfg.URL = strings.TrimRight(strings.TrimSpace(raw.URL), "/")
	if sub := strings.Trim(raw.Root, "/"); sub != "" && cfg.URL != "" && !strings.HasSuffix(cfg.URL, "/"+sub) {
		cfg.URL += "/" + sub
	}
	return cfg, nil
}

func cmpOr(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// location is the zone a date written without one is read in, which is the
// zone Hexo read it in: the site's, or the machine's when it names none.
func (c Config) location() (*time.Location, error) {
	if c.Timezone == "" {
		return time.Local, nil
	}
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return nil, fmt.Errorf("hexo: timezone %q in _config.yml is not a time zone", c.Timezone)
	}
	return loc, nil
}

// Report says what an import made.
type Report struct {
	Posts   int `json:"posts"`
	Drafts  int `json:"drafts"`
	Pages   int `json:"pages"`
	Files   int `json:"files"`
	Aliases int `json:"aliases"`

	// Tags lists the items whose text still holds Hexo tags, {% ... %},
	// which Kite does not read and prints as they are.
	Tags []string `json:"tags,omitempty"`

	// Skipped lists what was not imported, and why.
	Skipped []Skip `json:"skipped,omitempty"`
}

// Skip is a file left behind.
type Skip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// pending is an item about to be written, and the folder of files that goes
// into its bundle, if it has one.
type pending struct {
	item   *content.Content
	assets string
}

// Import reads the Hexo site at from into the project p.
func Import(ctx context.Context, from string, p *project.Project) (Report, error) {
	var report Report
	cfg, err := ReadConfig(from)
	if err != nil {
		return report, err
	}
	loc, err := cfg.location()
	if err != nil {
		return report, err
	}
	source := filepath.Join(from, filepath.FromSlash(cfg.SourceDir))
	if info, err := os.Stat(source); err != nil || !info.IsDir() {
		return report, fmt.Errorf("hexo: %s has no %s folder", from, cfg.SourceDir)
	}

	scan, err := p.Scanner().Scan()
	if err != nil {
		return report, err
	}
	slugs := make(map[string]bool)
	for _, e := range scan.Entries {
		slugs[string(e.Item.Kind)+"\x00"+e.Item.Slug] = true
	}

	im := &importer{cfg: cfg, loc: loc, from: from, source: source, slugs: slugs, report: &report}
	var items []pending
	for _, folder := range []string{"_posts", "_drafts"} {
		found, err := im.posts(folder)
		if err != nil {
			return report, err
		}
		items = append(items, found...)
	}
	pages, static, err := im.pages()
	if err != nil {
		return report, err
	}
	items = append(items, pages...)

	ops := make([]content.Op, len(items))
	for i, it := range items {
		ops[i] = content.PutContent{Content: it.item}
	}
	if len(ops) > 0 {
		if _, err := p.Writer().Apply(ctx, content.ChangeSet{Ops: ops, Message: "import from Hexo"}); err != nil {
			return report, err
		}
	}

	for _, it := range items {
		if it.assets == "" {
			continue
		}
		bundle := file.MediaDir(p.Types.Get(it.item.Kind), it.item.Locator)
		if bundle == "" {
			continue
		}
		if err := im.copyTree(it.assets, filepath.Join(p.Root, filepath.FromSlash(bundle))); err != nil {
			return report, err
		}
	}
	for _, rel := range static {
		target := filepath.Join(p.Root, "static", filepath.FromSlash(rel))
		if _, err := os.Stat(target); err == nil {
			report.Skipped = append(report.Skipped, Skip{Path: path.Join(cfg.SourceDir, rel), Reason: "static/" + rel + " already exists"})
			continue
		}
		if err := copyFile(filepath.Join(source, filepath.FromSlash(rel)), target); err != nil {
			return report, err
		}
		report.Files++
	}

	for _, it := range items {
		if hexoTag.MatchString(it.item.Body.Raw) {
			report.Tags = append(report.Tags, string(it.item.Locator))
		}
	}
	slices.Sort(report.Tags)
	return report, nil
}

type importer struct {
	cfg    Config
	loc    *time.Location
	from   string
	source string
	slugs  map[string]bool
	report *Report
}

// posts reads the posts in one of source's folders, _posts or _drafts. A
// folder beside a post with the post's name holds its files.
func (im *importer) posts(folder string) ([]pending, error) {
	dir := filepath.Join(im.source, folder)
	var sources []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		switch {
		case errors.Is(err, fs.ErrNotExist) && p == dir:
			return fs.SkipDir
		case err != nil:
			return err
		case strings.HasPrefix(d.Name(), ".") && p != dir:
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		case !d.IsDir() && isMarkdown(d.Name()):
			sources = append(sources, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	assetDirs := make(map[string]bool, len(sources))
	for _, s := range sources {
		assetDirs[strings.TrimSuffix(s, filepath.Ext(s))] = true
	}
	inAssets := func(p string) bool {
		for d := filepath.Dir(p); d != dir && len(d) > len(dir); d = filepath.Dir(d) {
			if assetDirs[d] {
				return true
			}
		}
		return false
	}

	var out []pending
	for _, s := range sources {
		if inAssets(s) {
			continue // a file of another post
		}
		it, err := im.post(dir, s, folder == "_drafts")
		if err != nil {
			im.skip(s, err.Error())
			continue
		}
		out = append(out, it)
	}
	im.skipLoose(dir, assetDirs, sources)
	return out, nil
}

// skipLoose reports files in a posts folder that belong to no post, which
// Hexo does not publish either.
func (im *importer) skipLoose(dir string, assetDirs map[string]bool, sources []string) {
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == dir {
			return nil
		}
		if d.IsDir() {
			if assetDirs[p] || strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !slices.Contains(sources, p) && !strings.HasPrefix(d.Name(), ".") {
			im.skip(p, "not a post, and in no post's folder")
		}
		return nil
	})
}

func (im *importer) post(dir, src string, draft bool) (pending, error) {
	fm, body, err := read(src)
	if err != nil {
		return pending{}, err
	}
	rel := filepath.ToSlash(strings.TrimSuffix(mustRel(dir, src), filepath.Ext(src)))
	name := path.Base(rel)

	item := &content.Content{
		Kind:   "post",
		Title:  cmpOr(fm.string("title"), name),
		Status: content.StatusPublished,
		Body:   content.Body{Format: content.FormatMarkdown, Raw: convertTags(body)},
	}
	if draft || fm.value("published") == false {
		item.Status = content.StatusDraft
	}
	item.Slug = im.claim("post", strings.Join(strings.Fields(name), "-"))

	published, err := im.date(fm, "date")
	if err != nil {
		return pending{}, err
	}
	if published.IsZero() {
		// Hexo dates a post that names no date by its file.
		info, err := os.Stat(src)
		if err != nil {
			return pending{}, err
		}
		published = info.ModTime().In(im.loc).Truncate(time.Second)
	}
	updated, err := im.date(fm, "updated")
	if err != nil {
		return pending{}, err
	}
	im.stamp(item, published, updated)

	item.Taxonomies = map[string][]string{}
	if tags := fm.strings("tags"); len(tags) > 0 {
		item.Taxonomies["tags"] = tags
	}
	if cats := fm.strings("categories"); len(cats) > 0 {
		item.Taxonomies["categories"] = cats
	}

	item.Meta = fm.params("post")
	// Only what Hexo published had an address to keep.
	if item.Status == content.StatusPublished {
		if custom := fm.string("permalink"); custom != "" {
			item.Aliases = append(item.Aliases, "/"+strings.TrimLeft(custom, "/"))
		} else if old, ok := im.permalink(rel, name, published.In(im.loc), fm); ok {
			item.Aliases = append(item.Aliases, old)
		}
	}
	im.report.Aliases += len(item.Aliases)
	if draft {
		im.report.Drafts++
	} else {
		im.report.Posts++
	}

	it := pending{item: item}
	if info, err := os.Stat(strings.TrimSuffix(src, filepath.Ext(src))); err == nil && info.IsDir() {
		it.assets = strings.TrimSuffix(src, filepath.Ext(src))
	}
	return it, nil
}

// pages reads every markdown file of source outside its _ folders as a page,
// and lists every other file there, which Hexo copies as it is.
func (im *importer) pages() ([]pending, []string, error) {
	var out []pending
	var static []string
	err := filepath.WalkDir(im.source, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == im.source {
			return nil
		}
		if strings.HasPrefix(d.Name(), "_") || strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel := filepath.ToSlash(mustRel(im.source, p))
		if !isMarkdown(d.Name()) {
			static = append(static, rel)
			return nil
		}
		it, err := im.page(p, rel)
		if err != nil {
			im.skip(p, err.Error())
			return nil
		}
		out = append(out, it)
		return nil
	})
	return out, static, err
}

func (im *importer) page(src, rel string) (pending, error) {
	fm, body, err := read(src)
	if err != nil {
		return pending{}, err
	}

	// Hexo published about/index.md at /about/, where Kite publishes a page
	// called about too, and about.md at /about.html. Its own index.md was
	// the home page, which is Kite's own.
	stem := strings.TrimSuffix(rel, path.Ext(rel))
	slug, index := strings.CutSuffix(stem, "/index")
	old := "/" + stem + ".html"
	switch {
	case stem == "index":
		slug, old = "home", ""
	case index:
		old = "/" + slug + "/"
	}

	item := &content.Content{
		Kind:   "page",
		Title:  cmpOr(fm.string("title"), path.Base(slug)),
		Status: content.StatusPublished,
		Body:   content.Body{Format: content.FormatMarkdown, Raw: convertTags(body)},
		Meta:   fm.params("page"),
	}
	if fm.value("published") == false {
		item.Status = content.StatusDraft
	}
	item.Slug = im.claim("page", slug)
	if old != "" && old != "/"+item.Slug+"/" {
		item.Aliases = []string{old}
		im.report.Aliases++
	}

	published, err := im.date(fm, "date")
	if err != nil {
		return pending{}, err
	}
	updated, err := im.date(fm, "updated")
	if err != nil {
		return pending{}, err
	}
	im.stamp(item, published, updated)
	im.report.Pages++
	return pending{item: item}, nil
}

// stamp dates an item: published when it went out, updated when it last
// changed, which is when it went out if nothing says otherwise. A draft has
// not gone out.
func (im *importer) stamp(item *content.Content, published, updated time.Time) {
	if updated.IsZero() {
		updated = published
	}
	item.CreatedAt, item.UpdatedAt = published.UTC(), updated.UTC()
	if item.Status == content.StatusPublished && !published.IsZero() {
		at := published.UTC()
		item.PublishedAt = &at
	}
}

// claim takes a free slug for an item of kind, base or base-2, base-3 and so
// on, among those the project already has and those claimed before it.
func (im *importer) claim(kind, base string) string {
	base = strings.Trim(base, "/")
	if base == "" {
		base = "untitled"
	}
	slug := base
	for i := 2; im.slugs[kind+"\x00"+slug]; i++ {
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	im.slugs[kind+"\x00"+slug] = true
	return slug
}

// date reads a date front matter names, in the zone Hexo read it in, or
// the zero time when it names none.
func (im *importer) date(fm frontMatter, key string) (time.Time, error) {
	raw, ok := fm.raw(key)
	if !ok || raw == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05Z07:00", "2006-01-02 15:04:05 -0700"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02 15:04", "2006/01/02 15:04:05", "2006-01-02", "2006/01/02"} {
		if t, err := time.ParseInLocation(layout, raw, im.loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("%s %q is not a date", key, raw)
}

// permalinkToken matches a token in a permalink pattern, such as :year.
var permalinkToken = regexp.MustCompile(`:[a-z_]+`)

// permalink is the address Hexo published a post at, from the site's pattern.
// A pattern holding a token that cannot be worked out here, such as :hash,
// gives none.
func (im *importer) permalink(rel, name string, date time.Time, fm frontMatter) (string, bool) {
	known := true
	link := permalinkToken.ReplaceAllStringFunc(im.cfg.Permalink, func(token string) string {
		switch key := token[1:]; key {
		case "year":
			return fmt.Sprintf("%04d", date.Year())
		case "month":
			return fmt.Sprintf("%02d", int(date.Month()))
		case "i_month":
			return fmt.Sprint(int(date.Month()))
		case "day":
			return fmt.Sprintf("%02d", date.Day())
		case "i_day":
			return fmt.Sprint(date.Day())
		case "hour":
			return fmt.Sprintf("%02d", date.Hour())
		case "minute":
			return fmt.Sprintf("%02d", date.Minute())
		case "second":
			return fmt.Sprintf("%02d", date.Second())
		case "title":
			return rel
		case "name":
			return name
		case "post_title":
			return cmpOr(fm.string("title"), name)
		case "category":
			if cats := fm.strings("categories"); len(cats) > 0 {
				return strings.Join(strings.Fields(cats[0]), "-")
			}
			if v, ok := im.cfg.PermalinkDefaults[key]; ok {
				return v
			}
			return "uncategorized"
		default:
			if v := fm.string(key); v != "" {
				return v
			}
			if v, ok := im.cfg.PermalinkDefaults[key]; ok {
				return v
			}
			known = false
			return token
		}
	})
	if !known {
		return "", false
	}
	return "/" + strings.TrimLeft(link, "/"), true
}

// hexoTag matches a Hexo tag, which Kite leaves as text.
var hexoTag = regexp.MustCompile(`\{%-?\s*[a-zA-Z_]+[^%]*%\}`)

// assetTag matches the tags that name one of a post's own files, which are
// written as the markdown that says the same.
var assetTag = regexp.MustCompile(`\{%\s*asset_(img|path|link)\s+(.*?)\s*%\}`)

func convertTags(body string) string {
	return assetTag.ReplaceAllStringFunc(body, func(tag string) string {
		m := assetTag.FindStringSubmatch(tag)
		args := splitArgs(m[2])
		if len(args) == 0 {
			return tag
		}
		name, title := args[0], strings.Join(args[1:], " ")
		dest := name
		if strings.ContainsAny(name, " ()") {
			dest = "<" + name + ">"
		}
		switch m[1] {
		case "img":
			return "![" + title + "](" + dest + ")"
		case "link":
			return "[" + cmpOr(title, name) + "](" + dest + ")"
		default:
			return name
		}
	})
}

// splitArgs splits a tag's arguments at spaces, keeping quoted ones whole.
func splitArgs(s string) []string {
	var out []string
	for s = strings.TrimSpace(s); s != ""; s = strings.TrimSpace(s) {
		if q := s[0]; q == '"' || q == '\'' {
			if end := strings.IndexByte(s[1:], q); end >= 0 {
				out = append(out, s[1:end+1])
				s = s[end+2:]
				continue
			}
		}
		word, rest, _ := strings.Cut(s, " ")
		out = append(out, word)
		s = rest
	}
	return out
}

func (im *importer) skip(p, reason string) {
	im.report.Skipped = append(im.report.Skipped, Skip{Path: filepath.ToSlash(mustRel(im.from, p)), Reason: reason})
}

// copyTree copies the files under from into to, keeping their folders.
func (im *importer) copyTree(from, to string) error {
	return filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return err
		}
		target := filepath.Join(to, mustRel(from, p))
		if _, err := os.Stat(target); err == nil {
			im.skip(p, filepath.ToSlash(target)+" already exists")
			return nil
		}
		if err := copyFile(p, target); err != nil {
			return err
		}
		im.report.Files++
		return nil
	})
}

func copyFile(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o644)
}

func isMarkdown(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown":
		return true
	}
	return false
}

func mustRel(base, p string) string {
	rel, err := filepath.Rel(base, p)
	if err != nil {
		return p
	}
	return rel
}
