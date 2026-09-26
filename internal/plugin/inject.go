package plugin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html/template"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/hook"
)

// injection is an Injection with its template parsed.
type injection struct {
	Injection
	tmpl *template.Template
}

// compile parses every injection's template, so that a broken one refuses
// the plugin when it loads rather than failing the first page it covers.
func compile(m Manifest) ([]injection, error) {
	out := make([]injection, 0, len(m.Inject))
	for i, in := range m.Inject {
		t, err := template.New(fmt.Sprintf("%s/inject/%d", m.ID, i+1)).
			Funcs(template.FuncMap{"asset": func(string) string { return "" }}).
			Parse(in.HTML)
		if err != nil {
			return nil, fmt.Errorf("plugin %s: inject %d: %w", m.ID, i+1, err)
		}
		out = append(out, injection{Injection: in, tmpl: t})
	}
	return out, nil
}

// Site is what a plugin is told about the site: injected code as .Site, a
// module as site.
type Site struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	BaseURL     string `json:"base_url"`
	Language    string `json:"language"`
}

// Page is what a plugin is told about a page: injected code as .Page, a
// module as page.
type Page struct {
	// URL is the page's address within the site, and Permalink the whole of
	// it, which is what a comment thread is usually keyed on.
	URL       string `json:"url"`
	Permalink string `json:"permalink"`

	// Kind is home, single, list, taxonomy, term or notFound.
	Kind string `json:"kind"`

	// Title is the page's: its item's on a single page, and the listing's
	// otherwise, where the page has one.
	Title string `json:"title,omitempty"`

	// The rest are the item's, on a single page: its id, which survives a
	// change of address, its content kind, its fields and its terms.
	ID          string              `json:"id,omitempty"`
	Type        string              `json:"type,omitempty"`
	Params      map[string]any      `json:"params,omitempty"`
	Taxonomies  map[string][]string `json:"taxonomies,omitempty"`
	PublishedAt *time.Time          `json:"published_at,omitempty"`
}

// Links tells a plugin where things are published.
type Links struct {
	// Asset is the address of a file of the plugin's own, by its path under
	// assets/.
	Asset func(name string) string
	// Absolute turns an address within the site into a whole one.
	Absolute func(url string) string
	// Item is the address of an item's page within the site.
	Item func(item *content.Content) string
}

// pageOf describes a page to a plugin.
func pageOf(url, kind string, item *content.Content, links Links) Page {
	page := Page{URL: url, Kind: kind}
	if links.Absolute != nil {
		page.Permalink = links.Absolute(url)
	}
	if item != nil {
		page.Title = item.Title
		page.ID, page.Type = string(item.ID), string(item.Kind)
		page.Params, page.Taxonomies, page.PublishedAt = item.Meta, item.Taxonomies, item.PublishedAt
	}
	return page
}

// Injector puts a plugin's code into the pages it covers.
type Injector struct {
	hook.Base
	settings map[string]any
	site     Site
	links    Links
	rules    []injection
}

// Injector returns the hook that injects the plugin's code into pages, with
// the settings a site gave it, or nil when it injects nothing.
func (p *Plugin) Injector(settings map[string]any, site Site, links Links) (*Injector, error) {
	if len(p.injections) == 0 {
		return nil, nil
	}
	resolved := p.Settings(settings)

	rules := make([]injection, len(p.injections))
	for i, in := range p.injections {
		t, err := in.tmpl.Clone()
		if err != nil {
			return nil, err
		}
		rules[i] = injection{Injection: in.Injection, tmpl: t.Funcs(template.FuncMap{"asset": links.Asset})}
	}

	// What the output depends on: the manifest, which holds the templates,
	// the settings and the site they are drawn with. Any of them changing
	// has to change every page the plugin touches.
	key, err := json.Marshal([]any{p.Manifest, resolved, site})
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(key)

	return &Injector{
		Base: hook.Base{
			HookName:    "plugin " + p.Manifest.ID,
			HookPhase:   hook.PhaseBuild,
			HookVersion: fmt.Sprintf("%x", sum),
		},
		settings: resolved,
		site:     site,
		links:    links,
		rules:    rules,
	}, nil
}

// covers reports whether a rule applies to a page, given the settings.
func (in *injection) covers(page Page, settings map[string]any) bool {
	if len(in.Pages) > 0 && !slices.Contains(in.Pages, page.Kind) {
		return false
	}
	if len(in.Kinds) > 0 && (page.Kind != "single" || !slices.Contains(in.Kinds, page.Type)) {
		return false
	}
	for key, want := range in.When {
		if !matches(settings[key], want) {
			return false
		}
	}
	return true
}

// matches compares a setting with what a rule wants of it, any of a list.
func matches(have, want any) bool {
	if list, ok := want.([]any); ok {
		return slices.ContainsFunc(list, func(w any) bool { return matches(have, w) })
	}
	return fmt.Sprint(have) == fmt.Sprint(want)
}

// TransformHTML adds the plugin's code to a page.
func (i *Injector) TransformHTML(_ context.Context, doc *hook.HTMLDoc) error {
	page := pageOf(doc.URL, doc.Kind, doc.Item, i.links)
	data := map[string]any{"Settings": i.settings, "Site": i.site, "Page": page}

	var head, body bytes.Buffer
	for _, rule := range i.rules {
		if !rule.covers(page, i.settings) {
			continue
		}
		into := &body
		if rule.At == AtHead {
			into = &head
		}
		if err := rule.tmpl.Execute(into, data); err != nil {
			return fmt.Errorf("%s: %w", rule.tmpl.Name(), err)
		}
		into.WriteByte('\n')
	}
	doc.HTML = insert(doc.HTML, head.String(), body.String())
	return nil
}

var (
	headEnd   = regexp.MustCompile(`(?i)</head\s*>`)
	bodyStart = regexp.MustCompile(`(?i)<body[\s>]`)
	bodyEnd   = regexp.MustCompile(`(?i)</body\s*>`)
)

// insert puts head code before the first </head> and body code before the
// last </body>. A page that leaves either tag out, which HTML allows, gets
// the head code before its body and the body code at its very end.
func insert(html, head, body string) string {
	if head != "" {
		switch at := headEnd.FindStringIndex(html); {
		case at != nil:
			html = html[:at[0]] + head + html[at[0]:]
		default:
			if at := bodyStart.FindStringIndex(html); at != nil {
				html = html[:at[0]] + head + html[at[0]:]
			} else {
				html = head + html
			}
		}
	}
	if body != "" {
		if all := bodyEnd.FindAllStringIndex(html, -1); len(all) > 0 {
			at := all[len(all)-1][0]
			html = html[:at] + body + html[at:]
		} else {
			html += body
		}
	}
	return html
}

// hostsIn finds the sites an injection loads from by name: addresses written
// into its src and href attributes. One built from a setting cannot be known
// until the setting is, and is not counted.
var hostsIn = regexp.MustCompile(`(?i)(?:src|href)\s*=\s*["']?(?:https?:)?//([a-z0-9.-]+\.[a-z]{2,})`)

// Hosts lists the other sites the plugin's code loads from, for the studio
// to say before it is turned on.
func (p *Plugin) Hosts() []string {
	var hosts []string
	for _, in := range p.Manifest.Inject {
		for _, m := range hostsIn.FindAllStringSubmatch(in.HTML, -1) {
			host := strings.ToLower(m[1])
			if !slices.Contains(hosts, host) {
				hosts = append(hosts, host)
			}
		}
	}
	slices.Sort(hosts)
	return hosts
}
