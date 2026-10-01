package apps

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/buildinfo"
)

// Bounds on what is read from the network. An archive is held to the bound
// Kite installs a package within.
const (
	maxIndex   = 16 << 20
	maxArchive = 64 << 20
)

// Client reads the index and fetches archives.
type Client struct {
	// URLs are where the index is read from, tried in order. The first names
	// the index, as kite.lock records it.
	URLs []string
	// CacheDir keeps the index and the archives fetched; nothing is kept
	// when it is empty.
	CacheDir string
	// MaxAge is how long a fetched index is used before it is asked for
	// again.
	MaxAge time.Duration
	HTTP   *http.Client
	Now    func() time.Time
}

// NewClient returns a client of the index at urls that keeps what it fetches
// in cacheDir.
func NewClient(urls []string, cacheDir string) *Client {
	return &Client{URLs: urls, CacheDir: cacheDir, MaxAge: time.Hour, HTTP: &http.Client{}, Now: time.Now}
}

// Source names the index the client reads.
func (c *Client) Source() string { return c.URLs[0] }

// Fetched says where an index came from.
type Fetched struct {
	URL string
	At  time.Time
	// Offline is set when no address answered, and the index is the copy
	// fetched at At; Err says why.
	Offline bool
	Err     error
}

// cacheMeta is what is kept beside a fetched index.
type cacheMeta struct {
	Source  string    `json:"source"`
	URL     string    `json:"url"`
	ETag    string    `json:"etag,omitempty"`
	Fetched time.Time `json:"fetched"`
}

// Index reads the index: the copy fetched within MaxAge unless refresh, or
// else from the first address that answers, asked with the copy's ETag so
// that an unchanged index is not sent again. When no address answers, the
// copy is used however old it is, and Fetched says so.
func (c *Client) Index(ctx context.Context, refresh bool) (*Index, Fetched, error) {
	meta, kept := c.cachedIndex()
	if kept != nil && !refresh && c.Now().Sub(meta.Fetched) < c.MaxAge {
		if ix, err := parseIndex(kept); err == nil {
			return ix, Fetched{URL: meta.URL, At: meta.Fetched}, nil
		}
	}
	var errs []error
	for _, addr := range c.URLs {
		etag := ""
		if kept != nil && meta.URL == addr {
			etag = meta.ETag
		}
		data, tag, err := c.fetchIndex(ctx, addr, etag)
		if err == nil && data == nil {
			data = kept
		}
		var ix *Index
		if err == nil {
			ix, err = parseIndex(data)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", addr, err))
			continue
		}
		now := c.Now()
		c.keepIndex(cacheMeta{Source: c.Source(), URL: addr, ETag: tag, Fetched: now}, data)
		return ix, Fetched{URL: addr, At: now}, nil
	}
	err := errors.Join(errs...)
	if kept != nil {
		if ix, perr := parseIndex(kept); perr == nil {
			return ix, Fetched{URL: meta.URL, At: meta.Fetched, Offline: true, Err: err}, nil
		}
	}
	return nil, Fetched{}, fmt.Errorf("the index of themes and plugins could not be read: %w", err)
}

// fetchIndex asks one address for the index. It returns no data and no
// error when the index has not changed since the copy tagged etag.
func (c *Client) fetchIndex(ctx context.Context, addr, etag string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := c.request(ctx, addr)
	if err != nil {
		return nil, "", err
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	tag := resp.Header.Get("ETag")
	switch {
	case resp.StatusCode == http.StatusNotModified && etag != "":
		if tag == "" {
			tag = etag
		}
		return nil, tag, nil
	case resp.StatusCode != http.StatusOK:
		return nil, "", fmt.Errorf("answered %s", resp.Status)
	}
	data, err := readAtMost(resp.Body, maxIndex)
	return data, tag, err
}

func parseIndex(data []byte) (*Index, error) {
	var ix Index
	if err := json.Unmarshal(data, &ix); err != nil {
		return nil, fmt.Errorf("not an index of themes and plugins: %w", err)
	}
	if ix.Format == 0 {
		return nil, errors.New("not an index of themes and plugins: it names no format")
	}
	return &ix, nil
}

// cachedIndex is the index kept from an earlier fetch of this client's
// index, if there is one.
func (c *Client) cachedIndex() (cacheMeta, []byte) {
	var meta cacheMeta
	if c.CacheDir == "" {
		return meta, nil
	}
	raw, err := os.ReadFile(filepath.Join(c.CacheDir, "index.meta.json"))
	if err != nil || json.Unmarshal(raw, &meta) != nil || meta.Source != c.Source() {
		return cacheMeta{}, nil
	}
	data, err := os.ReadFile(filepath.Join(c.CacheDir, "index.json"))
	if err != nil {
		return cacheMeta{}, nil
	}
	return meta, data
}

// keepIndex keeps a fetched index. A cache that cannot be written only means
// the next command asks again.
func (c *Client) keepIndex(meta cacheMeta, data []byte) {
	if c.CacheDir == "" {
		return
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return
	}
	if writeFile(filepath.Join(c.CacheDir, "index.json"), data) == nil {
		_ = writeFile(filepath.Join(c.CacheDir, "index.meta.json"), raw)
	}
}

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Archive fetches a release's archive from the first of its addresses that
// serves the bytes its checksum names, and keeps a copy named by the
// checksum, so that the same archive is fetched once.
func (c *Client) Archive(ctx context.Context, r *Release) ([]byte, error) {
	want := strings.ToLower(r.Archive.SHA256)
	if !sha256Hex.MatchString(want) {
		return nil, fmt.Errorf("the index gives version %s no sha256 to check its archive by", r.Version)
	}
	kept := ""
	if c.CacheDir != "" {
		kept = filepath.Join(c.CacheDir, "archives", want+".zip")
		if data, err := os.ReadFile(kept); err == nil && sum(data) == want {
			return data, nil
		}
	}
	var errs []error
	for _, addr := range r.Archive.URLs {
		data, err := c.fetchArchive(ctx, addr)
		switch {
		case err != nil:
		case sum(data) != want:
			err = errors.New("it is not the archive the index names: the sha256 differs")
		case r.Archive.Size > 0 && int64(len(data)) != r.Archive.Size:
			err = fmt.Errorf("it is %d bytes, and the index says %d", len(data), r.Archive.Size)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", addr, err))
			continue
		}
		if kept != "" {
			_ = writeFile(kept, data)
		}
		return data, nil
	}
	if len(errs) == 0 {
		return nil, fmt.Errorf("the index gives version %s no address to fetch it from", r.Version)
	}
	return nil, fmt.Errorf("the archive of version %s could not be fetched: %w", r.Version, errors.Join(errs...))
}

func (c *Client) fetchArchive(ctx context.Context, addr string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := c.request(ctx, addr)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("answered %s", resp.Status)
	}
	return readAtMost(resp.Body, maxArchive)
}

// do sends a request. Its address is left out of an error, since the
// caller names the address with every error it reports.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.HTTP.Do(req)
	var uerr *url.Error
	if errors.As(err, &uerr) {
		return nil, uerr.Err
	}
	return resp, err
}

func (c *Client) request(ctx context.Context, addr string) (*http.Request, error) {
	if !strings.HasPrefix(addr, "https://") && !strings.HasPrefix(addr, "http://") {
		return nil, errors.New("not an http or https address")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "kite/"+buildinfo.Version)
	return req, nil
}

// readAtMost reads a body, refusing one larger than limit however its size
// was declared.
func readAtMost(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("it is larger than %d MB", limit>>20)
	}
	return data, nil
}

// writeFile writes a file whole or not at all, so that a copy kept while
// another command reads it is never seen half written.
func writeFile(name string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(name), ".write-*")
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), name)
	}
	if err != nil {
		_ = os.Remove(f.Name())
	}
	return err
}

func sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
