package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kite-plus/kite/internal/version"
)

// updateCheckRepo is hard-coded so a self-hosted instance can never accidentally
// poll a fork — the upstream project IS the source of truth for "is there a
// newer release". If a fork wants to publish its own update channel they can
// patch this constant.
const (
	updateCheckOwner = "kite-plus"
	updateCheckRepo  = "kite"

	// GitHub's REST API endpoint for the latest non-prerelease release.
	// Returns 404 when the repo has no published releases yet.
	updateCheckLatestEndpoint = "https://api.github.com/repos/" + updateCheckOwner + "/" + updateCheckRepo + "/releases/latest"

	// In-memory cache TTL. Six hours keeps a typical multi-instance fleet
	// well below GitHub's 60/hour anonymous rate limit while still
	// surfacing new releases within a single workday.
	updateCheckCacheTTL = 6 * time.Hour

	// Per-request timeout for the upstream call. GitHub usually answers
	// in <500ms; bail out fast so the admin dashboard never hangs on a
	// flaky network.
	updateCheckHTTPTimeout = 5 * time.Second
)

// UpdateInfo is the wire-shape returned to the admin UI.
//
// Current may be a tag like "v1.0.0" or a sentinel like "dev" / something
// produced by `git describe --dirty`. The frontend only flags an update when
// HasUpdate is true; it surfaces Latest unconditionally so the admin can see
// what the upstream channel looks like even on dev builds.
type UpdateInfo struct {
	Current     string    `json:"current"`
	Latest      string    `json:"latest"`
	HasUpdate   bool      `json:"has_update"`
	PublishedAt time.Time `json:"published_at,omitempty"`
	HTMLURL     string    `json:"html_url,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
}

// UpdateCheckService polls GitHub's release API on demand and caches the
// answer in memory. There's no background goroutine — the cache is refreshed
// lazily when the first admin request after the TTL expires lands.
type UpdateCheckService struct {
	client *http.Client

	mu        sync.Mutex
	cached    *UpdateInfo
	cachedErr error
	cachedAt  time.Time
}

func NewUpdateCheckService() *UpdateCheckService {
	return &UpdateCheckService{
		client: &http.Client{Timeout: updateCheckHTTPTimeout},
	}
}

// Check returns the latest known UpdateInfo. When force is true the cache is
// bypassed; otherwise a value within the TTL is returned without a network
// round-trip. A failed upstream call is itself cached for one minute so a
// flapping admin dashboard doesn't pile requests onto an already-failing
// GitHub.
func (s *UpdateCheckService) Check(ctx context.Context, force bool) (*UpdateInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !force && s.cached != nil && time.Since(s.cachedAt) < updateCheckCacheTTL {
		return s.cached, nil
	}
	if !force && s.cachedErr != nil && time.Since(s.cachedAt) < time.Minute {
		return nil, s.cachedErr
	}

	info, err := s.fetch(ctx)
	s.cachedAt = time.Now()
	if err != nil {
		s.cachedErr = err
		return nil, err
	}
	s.cached = info
	s.cachedErr = nil
	return info, nil
}

func (s *UpdateCheckService) fetch(ctx context.Context) (*UpdateInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updateCheckLatestEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "kite-update-check")
	// Pin the API version so a future GitHub default change doesn't
	// silently shift the response shape under us.
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call github: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Repo has no published releases yet — surface this as a clean
		// "no upstream version" state rather than an error. The admin UI
		// just shows "current build" without the new-version badge.
		current := strings.TrimSpace(version.Get().Version)
		return &UpdateInfo{
			Current:   current,
			Latest:    "",
			HasUpdate: false,
			CheckedAt: time.Now(),
		}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github returned status %d", resp.StatusCode)
	}

	var payload struct {
		TagName     string    `json:"tag_name"`
		HTMLURL     string    `json:"html_url"`
		PublishedAt time.Time `json:"published_at"`
		Prerelease  bool      `json:"prerelease"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	current := strings.TrimSpace(version.Get().Version)
	latest := strings.TrimSpace(payload.TagName)

	return &UpdateInfo{
		Current:     current,
		Latest:      latest,
		HasUpdate:   isNewerVersion(latest, current),
		PublishedAt: payload.PublishedAt,
		HTMLURL:     payload.HTMLURL,
		CheckedAt:   time.Now(),
	}, nil
}

// isNewerVersion reports whether `latest` is strictly newer than `current` in
// semver terms. Both inputs may carry a leading "v" and either may be a dev
// build (e.g. "dev", "v1.0.0-5-gabc123-dirty"); when either side isn't a clean
// semver tag we fall back to "no update" so dev builds don't get spammed with
// upgrade banners.
func isNewerVersion(latest, current string) bool {
	if latest == "" || current == "" {
		return false
	}
	lp, lok := parseSemver(latest)
	cp, cok := parseSemver(current)
	if !lok || !cok {
		return false
	}
	for i := 0; i < 3; i++ {
		if lp[i] > cp[i] {
			return true
		}
		if lp[i] < cp[i] {
			return false
		}
	}
	return false
}

// parseSemver extracts the [major, minor, patch] tuple from a tag like
// "v1.2.3" or "1.2.3". Anything with a pre-release/build suffix is rejected
// so we only compare clean release tags — dirty dev builds shouldn't drive
// the upgrade banner.
func parseSemver(tag string) ([3]int, bool) {
	tag = strings.TrimPrefix(strings.TrimSpace(tag), "v")
	parts := strings.SplitN(tag, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i, p := range parts {
		// Reject anything that isn't a pure number — "1.0.0-rc1" or
		// "1.0.0+meta" both fall here. We don't try to compare
		// pre-release ordering; release-vs-release is enough for
		// the "is there a newer stable" check.
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, false
		}
		if n < 0 {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}

// ErrUpdateCheckUnavailable is the typed error callers can match on when they
// want to render a friendly "couldn't reach GitHub" message instead of a 500.
var ErrUpdateCheckUnavailable = errors.New("update check unavailable")
