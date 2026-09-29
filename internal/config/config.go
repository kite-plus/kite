// Package config loads kite.yaml.
//
// Configuration is an input to the core, and nothing outside this package
// reads the file: a module that parses its own corner of the configuration is
// a module whose defaults drift away from everyone else's.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	// Time zones are looked up by name on every system, Windows and a scratch
	// container included, rather than only where a zoneinfo database exists.
	_ "time/tzdata"

	"gopkg.in/yaml.v3"
)

// languageTag is the shape of a BCP 47 tag: a language, then optional
// subtags for script, region and variant.
//
// The shape is checked rather than the tag being looked up in a registry.
// What this is for is catching a typo or an emptied form field, and a
// registry would also refuse the private-use and grandfathered tags a real
// site is entitled to.
var languageTag = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

// WellFormedLanguage reports whether a language tag can be used.
//
// It is the lang attribute on every page, the prefix in every localized URL
// and the locale dates are written in. A wrong one does not fail: it produces
// a site that claims to be written in a language that does not exist, and an
// empty one produces <html lang="">.
func WellFormedLanguage(tag string) bool {
	return languageTag.MatchString(tag)
}

// DefaultLanguage is used when a project does not name one.
const DefaultLanguage = "en"

// Name is the configuration file that marks a project root.
const Name = "kite.yaml"

// Site describes the site itself.
type Site struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description,omitempty"`
	BaseURL     string `yaml:"baseURL"`
	Language    string `yaml:"language,omitempty"`
	Author      string `yaml:"author,omitempty"`

	// Keywords describe the site to search engines.
	Keywords Keywords `yaml:"keywords,omitempty"`

	// Timezone is the IANA zone dates are shown in. Left empty, a date is
	// shown in the zone it was written with, which for what the admin writes
	// is UTC: a post published just after midnight in Shanghai would read as
	// the day before.
	Timezone string `yaml:"timezone,omitempty"`

	// NoIndex asks search engines to leave the site out of their results.
	NoIndex bool `yaml:"noindex,omitempty"`

	// HeadHTML and FooterHTML are added to every page as written, for an
	// analytics or verification snippet that should survive a change of
	// theme.
	HeadHTML   string `yaml:"headHTML,omitempty"`
	FooterHTML string `yaml:"footerHTML,omitempty"`

	Params map[string]any `yaml:"params,omitempty"`
}

// Location is the time zone dates are shown in, or nil to show each date in
// the zone it was written with.
func (s Site) Location() (*time.Location, error) {
	switch s.Timezone {
	case "":
		return nil, nil
	case "Local":
		// Whatever zone the building machine is in: a CI runner and a
		// laptop would publish different dates for the same site.
		return nil, errors.New("the zone called Local depends on the machine; name a zone")
	}
	return time.LoadLocation(s.Timezone)
}

// Keywords is a list of keywords. Written by hand it may be one line,
// separated by commas, and it reads as the same list.
type Keywords []string

// UnmarshalYAML accepts a list or a line of comma separated words.
func (k *Keywords) UnmarshalYAML(node *yaml.Node) error {
	var words []string
	switch node.Kind {
	case yaml.ScalarNode:
		words = SplitKeywords(node.Value)
	case yaml.SequenceNode:
		if err := node.Decode(&words); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: keywords are a list or a line of words", node.Line)
	}
	*k = Keywords(cleanKeywords(words))
	return nil
}

// SplitKeywords splits a line at commas, ASCII or full width.
func SplitKeywords(line string) []string {
	return cleanKeywords(strings.FieldsFunc(line, func(r rune) bool {
		return r == ',' || r == '，' || r == '、'
	}))
}

// cleanKeywords trims each word and drops the empty ones and repeats.
func cleanKeywords(words []string) []string {
	out := make([]string, 0, len(words))
	seen := make(map[string]bool, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" || seen[word] {
			continue
		}
		seen[word] = true
		out = append(out, word)
	}
	return out
}

// Content selects the content store.
type Content struct {
	Store string `yaml:"store,omitempty"`
	Dir   string `yaml:"dir,omitempty"`
}

// Theme names the theme and carries its settings.
type Theme struct {
	Name     string         `yaml:"name,omitempty"`
	Settings map[string]any `yaml:"settings,omitempty"`
}

// Markdown configures the rendering pipeline.
type Markdown struct {
	// UnsafeHTML lets raw HTML through. It changes how every existing document
	// renders, so it is a decision made once per site.
	UnsafeHTML     bool   `yaml:"unsafeHTML,omitempty"`
	Typographer    bool   `yaml:"typographer,omitempty"`
	HardWraps      bool   `yaml:"hardWraps,omitempty"`
	HighlightTheme string `yaml:"highlightTheme,omitempty"`
}

// Build configures output.
type Build struct {
	Output   string `yaml:"output,omitempty"`
	URLStyle string `yaml:"urlStyle,omitempty"`

	// PageSize is how many items a page of a listing shows, unless the theme
	// or Pagination says otherwise for that kind of listing.
	PageSize int `yaml:"pageSize,omitempty"`

	// Pagination is how each kind of listing pages, by kind (home, list or
	// term): how many items a page shows, 0 for all of them on one. It
	// stands in for what the theme declares.
	Pagination map[string]int `yaml:"pagination,omitempty"`

	Minify    bool `yaml:"minify,omitempty"`
	Sitemap   bool `yaml:"sitemap,omitempty"`
	Feed      bool `yaml:"feed,omitempty"`
	FeedLimit int  `yaml:"feedLimit,omitempty"`

	// FeedAliases are more files the feed is written to beside rss.xml, from
	// the site's root, such as the index.xml a site moved from Hugo was
	// subscribed at. Feed readers do not follow a page that redirects.
	FeedAliases []string `yaml:"feedAliases,omitempty"`
}

// PageSizes is how each kind of listing pages when not by PageSize: what a
// theme declares, with the site's Pagination over it.
func (b Build) PageSizes(theme map[string]int) map[string]int {
	out := make(map[string]int, len(theme)+len(b.Pagination))
	maps.Copy(out, theme)
	maps.Copy(out, b.Pagination)
	return out
}

// OutputDir is the directory a build writes the site to. An absolute output
// is used as it stands, as kite build --output uses one; a relative one is
// taken from the project root.
func (b Build) OutputDir(root string) string {
	if filepath.IsAbs(b.Output) {
		return filepath.Clean(b.Output)
	}
	return filepath.Join(root, b.Output)
}

// Plugins chooses the plugins a site runs and holds their settings.
type Plugins struct {
	// Enabled lists the plugins that run, in the order they run, so that two
	// of them working on the same page always do so in the same sequence and
	// a build comes out the same every time.
	Enabled []string `yaml:"enabled,omitempty"`

	// Settings holds each plugin's settings by its id. A plugin that is
	// turned off keeps them, and finds them again when it is turned back on.
	Settings map[string]map[string]any `yaml:"settings,omitempty"`
}

// UnmarshalYAML also reads a plain list, as the plugins to run.
func (p *Plugins) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.SequenceNode {
		return node.Decode(&p.Enabled)
	}
	type plain Plugins
	return node.Decode((*plain)(p))
}

// pluginID is the shape of a plugin's id: the name of its directory under
// plugins/, and the key of its settings.
var pluginID = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidPluginID reports whether an id can name a plugin.
func ValidPluginID(id string) bool { return pluginID.MatchString(id) }

// Publish configures how content reaches its destination.
type Publish struct {
	Publisher string `yaml:"publisher,omitempty"`
	Branch    string `yaml:"branch,omitempty"`
	Message   string `yaml:"commitMessage,omitempty"`
}

// Config is a parsed kite.yaml.
type Config struct {
	Site     Site     `yaml:"site"`
	Content  Content  `yaml:"content,omitempty"`
	Theme    Theme    `yaml:"theme,omitempty"`
	Markdown Markdown `yaml:"markdown,omitempty"`
	Build    Build    `yaml:"build,omitempty"`
	Publish  Publish  `yaml:"publish,omitempty"`
	Plugins  Plugins  `yaml:"plugins,omitempty"`
}

// validFeedAliases checks that each feed alias is a file of its own within
// the site that no other output is written to.
func validFeedAliases(aliases []string) error {
	seen := make(map[string]bool, len(aliases))
	for _, p := range aliases {
		ext := strings.ToLower(path.Ext(p))
		switch {
		case p == "" || strings.HasSuffix(p, "/") || strings.Contains(p, ".."):
			return fmt.Errorf("config: build.feedAliases: %q is not a file within the site", p)
		case ext == "" || ext == ".html" || ext == ".htm":
			return fmt.Errorf("config: build.feedAliases: %q is not a feed's file name (want something like index.xml)", p)
		case p == "rss.xml" || p == "sitemap.xml":
			return fmt.Errorf("config: build.feedAliases: Kite writes %s itself", p)
		case seen[p]:
			return fmt.Errorf("config: build.feedAliases lists %q twice", p)
		}
		seen[p] = true
	}
	return nil
}

// Default returns the configuration of a site that specifies nothing.
func Default() *Config {
	return &Config{
		Site:     Site{Title: "A Kite site", Language: "en"},
		Content:  Content{Store: "file", Dir: "content"},
		Theme:    Theme{Name: "default"},
		Markdown: Markdown{HighlightTheme: "github"},
		Build: Build{
			Output:    "public",
			URLStyle:  "directory",
			PageSize:  10,
			Sitemap:   true,
			Feed:      true,
			FeedLimit: 20,
		},
		Publish: Publish{Publisher: "git", Branch: "main"},
	}
}

// Load reads kite.yaml from a project root and applies defaults and
// environment overrides.
func Load(root string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(filepath.Join(root, Name))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, fmt.Errorf("config: %s not found in %s", Name, root)
	case err != nil:
		return nil, err
	}

	// Unmarshalling over the defaults leaves unmentioned keys at their default
	// value, so a minimal config file stays valid as options are added.
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", Name, err)
	}

	applyEnv(cfg)
	cfg.normalize()
	return cfg, cfg.Validate()
}

// applyEnv lets KITE_ variables override the file, which is how a CI job
// points a build at a different base URL without editing the repository.
func applyEnv(cfg *Config) {
	set := func(key string, apply func(string)) {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			apply(v)
		}
	}
	set("KITE_SITE_TITLE", func(v string) { cfg.Site.Title = v })
	set("KITE_SITE_BASEURL", func(v string) { cfg.Site.BaseURL = v })
	set("KITE_SITE_LANGUAGE", func(v string) { cfg.Site.Language = v })
	set("KITE_THEME", func(v string) { cfg.Theme.Name = v })
	set("KITE_BUILD_OUTPUT", func(v string) { cfg.Build.Output = v })
	set("KITE_BUILD_URLSTYLE", func(v string) { cfg.Build.URLStyle = v })
	set("KITE_BUILD_PAGESIZE", func(v string) {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Build.PageSize = n
		}
	})
}

func (c *Config) normalize() {
	c.Site.BaseURL = strings.TrimRight(c.Site.BaseURL, "/")
	if c.Build.PageSize <= 0 {
		c.Build.PageSize = 10
	}
	if c.Build.FeedLimit <= 0 {
		c.Build.FeedLimit = 20
	}
	for i, p := range c.Build.FeedAliases {
		c.Build.FeedAliases[i] = strings.TrimLeft(strings.TrimSpace(p), "/")
	}
	if c.Build.Output == "" {
		c.Build.Output = "public"
	}
	if c.Build.URLStyle == "" {
		c.Build.URLStyle = "directory"
	}
	if c.Theme.Name == "" {
		c.Theme.Name = "default"
	}
	// An absent language is not a broken one. Defaulting it here means a
	// project that never mentioned a language keeps working, and an emptied
	// form field falls back rather than producing <html lang="">.
	if c.Site.Language == "" {
		c.Site.Language = DefaultLanguage
	}
}

// Listings are the kinds of listing whose pages a site or a theme can size.
var Listings = []string{"home", "list", "term"}

// ValidPagination checks the page size of each kind of listing.
func ValidPagination(sizes map[string]int) error {
	for _, kind := range slices.Sorted(maps.Keys(sizes)) {
		switch {
		case !slices.Contains(Listings, kind):
			return fmt.Errorf("%q is not a kind of listing (want home, list or term)", kind)
		case sizes[kind] < 0:
			return fmt.Errorf("%s: %d is not a page size (want 0 for all on one page, or more)", kind, sizes[kind])
		}
	}
	return nil
}

// Validate rejects configurations that would fail later in a confusing place.
func (c *Config) Validate() error {
	switch c.Build.URLStyle {
	case "directory", "extension":
	default:
		return fmt.Errorf("config: unknown build.urlStyle %q (want directory or extension)", c.Build.URLStyle)
	}
	if c.Content.Store != "" && c.Content.Store != "file" {
		return fmt.Errorf("config: content.store %q is not implemented yet (only file)", c.Content.Store)
	}
	if !WellFormedLanguage(c.Site.Language) {
		return fmt.Errorf("config: site.language %q is not a language tag (want something like en or zh-CN)",
			c.Site.Language)
	}
	if _, err := c.Site.Location(); err != nil {
		return fmt.Errorf("config: site.timezone %q is not a time zone (want something like Asia/Shanghai or UTC)",
			c.Site.Timezone)
	}
	if !filepath.IsAbs(c.Build.Output) && !filepath.IsLocal(c.Build.Output) {
		return fmt.Errorf("config: build.output %q leaves the project (want a path inside it, or an absolute one)",
			c.Build.Output)
	}
	if err := validFeedAliases(c.Build.FeedAliases); err != nil {
		return err
	}
	if err := ValidPagination(c.Build.Pagination); err != nil {
		return fmt.Errorf("config: build.pagination: %w", err)
	}
	seen := make(map[string]bool, len(c.Plugins.Enabled))
	for _, id := range c.Plugins.Enabled {
		switch {
		case !ValidPluginID(id):
			return fmt.Errorf("config: plugins.enabled: %q is not a plugin id (want lowercase words joined by -)", id)
		case seen[id]:
			return fmt.Errorf("config: plugins.enabled lists %q twice", id)
		}
		seen[id] = true
	}
	return nil
}
