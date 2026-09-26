package api

import (
	"errors"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/kite-plus/kite/internal/auth"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/setup"
)

// SetupState is what a browser is told about a server nobody has configured.
//
// It carries what the project currently says about itself so that the form
// starts from the real values rather than from blanks, and nothing else: a
// server waiting to be set up is a server anyone can reach, and it has no
// business describing its content until it has an owner.
type SetupState struct {
	// Required is whether setup still has to happen. When it is false, the
	// rest of this is absent and the client goes to the studio.
	Required bool `json:"required"`

	// Site is what the project is configured with now, to fill the form in.
	Site *SiteSettings `json:"site,omitempty"`

	// User is the account name to suggest.
	User string `json:"user,omitempty"`

	// MinPasswordLength lets the form refuse a short password before asking
	// the server to.
	MinPasswordLength int `json:"min_password_length,omitempty"`

	// NewSite is a folder with no site in it yet. Setup then creates the site
	// and asks for no account, since such a server listens on this machine
	// only; the form is the site's half alone.
	NewSite bool `json:"new_site,omitempty"`
}

// SetupRequest finishes the first run: it describes the site and creates the
// one account that will guard it.
type SetupRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`

	Site SiteSettings `json:"site"`
}

// setupPaths answer while a server is waiting to be set up. Everything else
// is refused, including reads: an unconfigured server is one anyone can
// reach, and its drafts are not a welcome message.
var setupPaths = map[string]bool{
	OpenAPIPath:     true,
	"/setup":        true,
	"/auth/session": true,
}

// handleSetupState reports whether this server still needs setting up.
func (s *Server) handleSetupState(w http.ResponseWriter, _ *http.Request) {
	if !s.setup.Pending() {
		writeJSON(w, http.StatusOK, SetupState{Required: false})
		return
	}
	if s.setup.CreatesSite() {
		d := s.setup.Defaults()
		writeJSON(w, http.StatusOK, SetupState{
			Required: true,
			NewSite:  true,
			Site:     &SiteSettings{Title: d.Title, Description: d.Description, BaseURL: d.BaseURL, Language: d.Language},
		})
		return
	}

	site := s.src().Site
	writeJSON(w, http.StatusOK, SetupState{
		Required: true,
		Site: &SiteSettings{
			Title:       site.Title,
			Description: site.Description,
			BaseURL:     site.BaseURL,
			Language:    site.Language,
		},
		User:              "admin",
		MinPasswordLength: auth.MinPasswordLength,
	})
}

// handleSetup describes the site and creates its account.
//
// The settings are written before the account, because writing the account is
// what ends the flow: a failure anywhere leaves setup still open and the form
// still usable, rather than a server nobody can sign in to.
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if !s.setup.Pending() {
		fail(w, http.StatusConflict, CodeAlreadySetUp,
			"this server is already set up; sign in instead")
		return
	}

	req, ok := decodeJSON[SetupRequest](s, w, r)
	if !ok {
		return
	}
	if s.setup.CreatesSite() {
		s.createSite(w, req.Site)
		return
	}

	view := s.src()
	if !s.writable(w, view) {
		return
	}

	values, valid := setupSettings(w, req.Site)
	if !valid {
		return
	}
	user := strings.TrimSpace(req.User)
	if user == "" {
		user = "admin"
	}
	// Checked here as well as in the store, so that a password too short to
	// keep is refused before the site's own settings have been rewritten.
	if len([]rune(req.Password)) < auth.MinPasswordLength {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "password",
			"a password needs at least "+strconv.Itoa(auth.MinPasswordLength)+" characters")
		return
	}

	if _, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{content.PutSettings{Values: values}},
		Message: "setup: describe the site",
	}); err != nil {
		s.failWrite(w, view, r, err)
		return
	}

	account, err := s.setup.Complete(user, req.Password)
	if err != nil {
		s.failSetup(w, err)
		return
	}
	s.log.Info("set up", "user", account.User(), "from", account.Source())

	// Signed in on the spot. Asking for the password again, one field below
	// where it was just chosen, would be ceremony rather than a check.
	session, err := s.auth.Start(w, r)
	if err != nil {
		s.failErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, SessionInfo{
		Required:      true,
		Authenticated: true,
		User:          session.User,
		ExpiresAt:     session.Expires,
	})
}

// createSite finishes the first run of a folder with no site in it: the
// answers become the project, checked as the settings page checks them.
func (s *Server) createSite(w http.ResponseWriter, answers SiteSettings) {
	if _, valid := setupSettings(w, answers); !valid {
		return
	}
	site := setup.Site{
		Title:       strings.TrimSpace(answers.Title),
		Description: strings.TrimSpace(answers.Description),
		BaseURL:     strings.TrimSpace(answers.BaseURL),
		Language:    strings.TrimSpace(answers.Language),
	}
	if err := s.setup.CreateSite(site); err != nil {
		s.failSetup(w, err)
		return
	}
	s.log.Info("created the site", "title", site.Title)
	// Nothing guards the new studio: it is on this machine only.
	writeJSON(w, http.StatusOK, SessionInfo{Required: false})
}

// setupSettings turns the form into configuration values, refusing the ones
// the settings endpoint would refuse and for the same reasons.
func setupSettings(w http.ResponseWriter, site SiteSettings) (map[string]any, bool) {
	values := map[string]any{
		"site.title":    strings.TrimSpace(site.Title),
		"site.baseURL":  strings.TrimSpace(site.BaseURL),
		"site.language": strings.TrimSpace(site.Language),
	}
	// A description nobody wrote is left out rather than written empty: a
	// key set to "" in the configuration file reads as a decision, and this
	// one would be an unanswered optional question.
	if about := strings.TrimSpace(site.Description); about != "" {
		values["site.description"] = about
	}

	// Sorted, so that a form with two bad fields always names the same one
	// first rather than a different one on each attempt.
	for _, path := range slices.Sorted(maps.Keys(values)) {
		if problem := checkSetting(path, values[path]); problem != "" {
			failField(w, http.StatusBadRequest, CodeInvalidRequest, path, problem)
			return nil, false
		}
	}
	return values, true
}

// failSetup maps a refusal from the flow onto a response.
func (s *Server) failSetup(w http.ResponseWriter, err error) {
	if errors.Is(err, setup.ErrDone) || errors.Is(err, auth.ErrAlreadyConfigured) {
		fail(w, http.StatusConflict, CodeAlreadySetUp,
			"this server is already set up; sign in instead")
		return
	}
	s.failErr(w, err)
}
