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

	"aead.dev/minisign"

	"github.com/kite-plus/kite/internal/buildinfo"
)

// ErrUnreachable is reported, wrapped, when none of the addresses of the
// index or of an archive served it.
var ErrUnreachable = errors.New("unreachable")

// unreachable is an error ErrUnreachable stands for, worded on its own.
type unreachable struct{ error }

func (unreachable) Is(target error) bool { return target == ErrUnreachable }
func (u unreachable) Unwrap() error      { return u.error }

// answered is a status other than the one a request was after.
type answered string

func (a answered) Error() string { return "answered " + string(a) }

// Bounds on what is read from the network. An archive is held to the bound
// Kite installs a package within.
const (
	maxIndex     = 16 << 20
	maxSignature = 16 << 10
	maxArchive   = 64 << 20
	maxPicture   = 8 << 20
	maxIcon      = 64 << 10
)

// Client reads the index and fetches archives.
type Client struct {
	// URLs are where the index is read from, tried in order. The first names
	// the index, as kite.lock records it.
	URLs []string
	// Key is the public key the index has to be signed with. An index it did
	// not sign is refused, and so is an archive the index does not name.
	Key minisign.PublicKey
	// CacheDir keeps the index and the archives fetched; nothing is kept
	// when it is empty.
	CacheDir string
	// MaxAge is how long a fetched index is used before it is asked for
	// again.
	MaxAge time.Duration
	HTTP   *http.Client
	Now    func() time.Time
}

// NewClient returns a client of the index at urls, signed with key, that
// keeps what it fetches in cacheDir.
func NewClient(urls []string, cacheDir string, key minisign.PublicKey) *Client {
	return &Client{URLs: urls, Key: key, CacheDir: cacheDir, MaxAge: time.Hour, HTTP: &http.Client{}, Now: time.Now}
}

// Source names the index the client reads.
func (c *Client) Source() string { return c.URLs[0] }

// Fetched says where an index came from.
type Fetched struct {
	URL string
	At  time.Time
	// Offline is set when no address answered with an index that could be
	// used, and the index is the copy fetched at At; Err says why.
	Offline bool
	Err     error
}

// cacheMeta is what is kept beside a fetched index.
type cacheMeta struct {
	Source  string    `json:"source"`
	URL     string    `json:"url"`
	ETag    string    `json:"etag,omitempty"`
	Fetched time.Time `json:"fetched"`
	// Newest is when the newest index the client accepted was generated.
	Newest string `json:"newest,omitempty"`
}

// Index reads the index: the copy fetched within MaxAge unless refresh, or
// else from the first address that answers with an index its key signed,
// asked with the copy's ETag so that an unchanged index is not sent again.
// An index generated before the newest one accepted is refused. When no
// address answers with one, the copy is used however old it is, and Fetched
// says so.
func (c *Client) Index(ctx context.Context, refresh bool) (*Index, Fetched, error) {
	meta, kept, keptSig := c.cachedIndex()
	if kept != nil && !refresh && c.Now().Sub(meta.Fetched) < c.MaxAge {
		if ix, err := parseIndex(kept); err == nil {
			return ix, Fetched{URL: meta.URL, At: meta.Fetched}, nil
		}
	}
	var errs []error
	distrusted := false
	for _, addr := range c.URLs {
		etag := ""
		if kept != nil && meta.URL == addr {
			etag = meta.ETag
		}
		data, tag, err := c.fetchIndex(ctx, addr, etag)
		var sig []byte
		switch {
		case err != nil:
		case data == nil:
			data, sig = kept, keptSig
		default:
			if sig, err = c.fetch(ctx, addr+".minisig", maxSignature, 20*time.Second); err != nil {
				err = unsigned(err)
			} else {
				err = c.verify(data, sig)
			}
		}
		var ix *Index
		if err == nil {
			ix, err = parseIndex(data)
		}
		if err == nil && older(ix.Generated, meta.Newest) {
			err = untrusted{fmt.Errorf("it was generated at %s, before the index already seen, of %s", ix.Generated, meta.Newest)}
		}
		if err != nil {
			distrusted = distrusted || errors.Is(err, ErrUntrusted)
			errs = append(errs, fmt.Errorf("%s: %w", addr, err))
			continue
		}
		now := c.Now()
		newest := meta.Newest
		if newest == "" || older(newest, ix.Generated) {
			newest = ix.Generated
		}
		c.keepIndex(cacheMeta{Source: c.Source(), URL: addr, ETag: tag, Fetched: now, Newest: newest}, data, sig)
		return ix, Fetched{URL: addr, At: now}, nil
	}
	err := errors.Join(errs...)
	if kept != nil {
		if ix, perr := parseIndex(kept); perr == nil {
			return ix, Fetched{URL: meta.URL, At: meta.Fetched, Offline: true, Err: err}, nil
		}
	}
	err = fmt.Errorf("the index of themes and plugins could not be read: %w", err)
	if distrusted {
		return nil, Fetched{}, untrusted{err}
	}
	return nil, Fetched{}, unreachable{err}
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
		return nil, "", answered(resp.Status)
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
// index, with its signature, if there is one its key signed. A copy that is
// not, as one kept by a Kite that did not check signatures, is fetched
// again, though the meta it leaves still says how new an index has been.
func (c *Client) cachedIndex() (cacheMeta, []byte, []byte) {
	var meta cacheMeta
	if c.CacheDir == "" {
		return meta, nil, nil
	}
	raw, err := os.ReadFile(filepath.Join(c.CacheDir, "index.meta.json"))
	if err != nil || json.Unmarshal(raw, &meta) != nil || meta.Source != c.Source() {
		return cacheMeta{}, nil, nil
	}
	data, err := os.ReadFile(filepath.Join(c.CacheDir, "index.json"))
	if err != nil {
		return meta, nil, nil
	}
	sig, err := os.ReadFile(filepath.Join(c.CacheDir, "index.json.minisig"))
	if err != nil || c.verify(data, sig) != nil {
		return meta, nil, nil
	}
	return meta, data, sig
}

// keepIndex keeps a fetched index with its signature. A cache that cannot be
// written only means the next command asks again.
func (c *Client) keepIndex(meta cacheMeta, data, sig []byte) {
	if c.CacheDir == "" {
		return
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return
	}
	if writeFile(filepath.Join(c.CacheDir, "index.json"), data) == nil &&
		writeFile(filepath.Join(c.CacheDir, "index.json.minisig"), sig) == nil {
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
	return nil, unreachable{fmt.Errorf("the archive of version %s could not be fetched: %w", r.Version, errors.Join(errs...))}
}

// Picture fetches a picture the index points to, such as a theme's
// screenshot, and keeps it named by its address, which for a listed version
// never serves anything else. Only a picture is returned: bytes that are not
// one, such as a page, are refused, since they are served on to a browser.
func (c *Client) Picture(ctx context.Context, addr string) ([]byte, string, error) {
	return c.image(ctx, addr, maxPicture, picture)
}

// Icon fetches a package's icon as Picture fetches a picture, and takes an
// SVG too when there is nothing in it to run or to fetch; whoever serves it
// still keeps a browser from running it.
func (c *Client) Icon(ctx context.Context, addr string) ([]byte, string, error) {
	return c.image(ctx, addr, maxIcon, func(data []byte) (string, bool) {
		if ctype, ok := picture(data); ok {
			return ctype, true
		}
		return "image/svg+xml", harmlessSVG(data)
	})
}

// image fetches what kind says is an image, keeping a copy by its address.
func (c *Client) image(ctx context.Context, addr string, limit int64, kind func([]byte) (string, bool)) ([]byte, string, error) {
	kept := ""
	if c.CacheDir != "" {
		h := sha256.Sum256([]byte(addr))
		kept = filepath.Join(c.CacheDir, "pictures", hex.EncodeToString(h[:16]))
		if data, err := os.ReadFile(kept); err == nil {
			if ctype, ok := kind(data); ok {
				return data, ctype, nil
			}
		}
	}
	data, err := c.fetch(ctx, addr, limit, 30*time.Second)
	if err != nil {
		return nil, "", unreachable{fmt.Errorf("%s: %w", addr, err)}
	}
	ctype, ok := kind(data)
	if !ok {
		return nil, "", fmt.Errorf("%s is not a picture", addr)
	}
	if kept != "" {
		_ = writeFile(kept, data)
	}
	return data, ctype, nil
}

// picture is the type of an image, read from its bytes. SVG is not one: it
// can carry scripts.
func picture(data []byte) (string, bool) {
	ctype := http.DetectContentType(data)
	switch ctype {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/avif":
		return ctype, true
	}
	return "", false
}

func (c *Client) fetchArchive(ctx context.Context, addr string) ([]byte, error) {
	return c.fetch(ctx, addr, maxArchive, 2*time.Minute)
}

func (c *Client) fetch(ctx context.Context, addr string, limit int64, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
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
		return nil, answered(resp.Status)
	}
	return readAtMost(resp.Body, limit)
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
		if limit < 1<<20 {
			return nil, fmt.Errorf("it is larger than %d KB", limit>>10)
		}
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
