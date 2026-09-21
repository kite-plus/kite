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
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

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
	Title       string         `yaml:"title"`
	Description string         `yaml:"description,omitempty"`
	BaseURL     string         `yaml:"baseURL"`
	Language    string         `yaml:"language,omitempty"`
	Params      map[string]any `yaml:"params,omitempty"`
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
	Output    string `yaml:"output,omitempty"`
	URLStyle  string `yaml:"urlStyle,omitempty"`
	PageSize  int    `yaml:"pageSize,omitempty"`
	Minify    bool   `yaml:"minify,omitempty"`
	Sitemap   bool   `yaml:"sitemap,omitempty"`
	Feed      bool   `yaml:"feed,omitempty"`
	FeedLimit int    `yaml:"feedLimit,omitempty"`
}

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
	Plugins  []string `yaml:"plugins,omitempty"`
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
	if strings.Contains(c.Build.Output, "..") {
		return fmt.Errorf("config: build.output must stay inside the project")
	}
	return nil
}
