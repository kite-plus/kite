package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/auth"
)

// AccountInfo is what the studio knows about the person using it, and how
// they sign in.
type AccountInfo struct {
	// Protected reports whether the studio asks for a password.
	Protected bool `json:"protected"`

	// User is the name that signs in, absent while nothing guards the studio.
	User string `json:"user,omitempty"`

	// Source is where the account comes from: "file" for one stored in the
	// project, "environment" for one a deployment sets with KITE_ADMIN_USER
	// and KITE_ADMIN_PASSWORD, which is changed where it is set.
	Source string `json:"source,omitempty"`

	// Editable reports whether the name and password can be set from here:
	// an open studio given them, or a stored account changed.
	Editable bool `json:"editable"`

	// Removable reports whether the password can be taken away from here,
	// which only a server nobody else can reach allows.
	Removable bool `json:"removable"`

	// ProfileEditable reports whether the name and picture can be changed.
	ProfileEditable bool `json:"profile_editable"`

	MinPasswordLength int `json:"min_password_length"`

	Profile Profile `json:"profile"`

	// Avatar is where the picture is. It changes whenever the picture does,
	// so a browser never shows an old one from its cache. Absent when there
	// is none.
	Avatar string `json:"avatar,omitempty"`

	// Session is the caller's own, absent on a studio that needs none.
	Session *AccountSession `json:"session,omitempty"`
}

// Profile is how the person using the studio is shown in it. It is never
// published: the site's author is site.author.
type Profile struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AccountSession describes the caller's session.
type AccountSession struct {
	ExpiresAt time.Time `json:"expires_at"`
	// Remembered is whether it outlives the browser window.
	Remembered bool `json:"remembered"`
}

// CredentialsChange sets how the studio is signed in to.
type CredentialsChange struct {
	// CurrentPassword confirms the change. It is required whenever there is
	// a password to confirm, and ignored while there is none.
	CurrentPassword string `json:"current_password,omitempty"`

	// User is the name to sign in with. Empty keeps the current one, or is
	// admin for a studio that has none yet.
	User string `json:"user,omitempty"`

	// Password is the new password. Empty keeps the current one; a studio
	// that has none yet needs one.
	Password string `json:"password,omitempty"`
}

// PasswordConfirmation confirms a change with the current password.
type PasswordConfirmation struct {
	CurrentPassword string `json:"current_password"`
}

// handleAccount describes the person using the studio.
func (s *Server) handleAccount(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.accountInfo(r, nil))
}

// accountInfo builds the description, with session as the caller's own when
// it was just issued and so is not yet in the request.
func (s *Server) accountInfo(r *http.Request, session *auth.Session) AccountInfo {
	account := s.auth.Account()
	writable := s.src().Writer != nil

	info := AccountInfo{
		Protected:         account != nil,
		ProfileEditable:   s.keeper != nil && writable,
		MinPasswordLength: auth.MinPasswordLength,
	}
	info.Editable = info.ProfileEditable
	if account != nil {
		info.User = account.User()
		info.Source = "file"
		if account.Source() != auth.File {
			info.Source = "environment"
			info.Editable = false
		}
		info.Removable = info.Editable && s.keeper.MayOpen()
	}

	if s.keeper != nil {
		profile, err := s.keeper.Profile()
		if err != nil {
			// A profile nobody can read is shown as none rather than taking
			// the studio's menu down with it; saving one writes it afresh.
			s.log.Error("read profile", "err", err)
		}
		info.Profile = Profile{Name: profile.Name, Email: profile.Email}

		if data, _, err := s.keeper.Avatar(); err == nil {
			info.Avatar = Prefix + "/account/avatar?v=" + version(data)
		} else if !errors.Is(err, auth.ErrNoAvatar) {
			s.log.Error("read avatar", "err", err)
		}
	}

	if session == nil {
		if current, err := s.auth.Session(r); err == nil {
			session = &current
		}
	}
	if session != nil && account != nil {
		info.Session = &AccountSession{ExpiresAt: session.Expires, Remembered: session.Remember}
	}
	return info
}

// version names a picture by what it holds.
func version(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:6])
}

// account answers for a server that keeps no account, which a server built
// without one is: every change here needs somewhere to write it.
func (s *Server) account(w http.ResponseWriter) (*auth.Keeper, bool) {
	if s.keeper == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server does not keep an account")
		return nil, false
	}
	return s.keeper, s.writable(w, s.src())
}

// handleUpdateProfile changes how the person using the studio is shown.
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	keeper, ok := s.account(w)
	if !ok {
		return
	}
	req, ok := decodeJSON[Profile](s, w, r)
	if !ok {
		return
	}
	if _, err := keeper.SaveProfile(auth.Profile{Name: req.Name, Email: req.Email}); err != nil {
		s.failAccount(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.accountInfo(r, nil))
}

// handleAvatar serves the picture.
func (s *Server) handleAvatar(w http.ResponseWriter, r *http.Request) {
	if s.keeper == nil {
		fail(w, http.StatusNotFound, CodeNotFound, "no picture has been chosen")
		return
	}
	data, kind, err := s.keeper.Avatar()
	if errors.Is(err, auth.ErrNoAvatar) {
		fail(w, http.StatusNotFound, CodeNotFound, "no picture has been chosen")
		return
	}
	if err != nil {
		s.failErr(w, err)
		return
	}

	h := w.Header()
	h.Set("Content-Type", kind)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "default-src 'none'")
	h.Set("ETag", `"`+version(data)+`"`)
	// The address names the picture by its content, so what a browser keeps
	// under it is never out of date; private, because it is somebody's.
	h.Set("Cache-Control", "private, max-age=31536000, immutable")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(data))
}

// handleUploadAvatar stores a new picture.
func (s *Server) handleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	keeper, ok := s.account(w)
	if !ok {
		return
	}

	// Bounded before it is parsed, so an oversized upload is refused rather
	// than spooled to disk first.
	r.Body = http.MaxBytesReader(w, r.Body, auth.MaxAvatarSize+64<<10)
	file, _, err := r.FormFile("file")
	if err != nil {
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			fail(w, http.StatusRequestEntityTooLarge, CodeInvalidRequest, "the picture is too large")
			return
		}
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "file",
			"send the picture as multipart form data under the name \"file\"")
		return
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, auth.MaxAvatarSize+1))
	if err != nil {
		s.failErr(w, err)
		return
	}
	if len(data) > auth.MaxAvatarSize {
		fail(w, http.StatusRequestEntityTooLarge, CodeInvalidRequest, "the picture is too large")
		return
	}
	if _, err := keeper.SaveAvatar(data); err != nil {
		if bad, ok := errors.AsType[*auth.ProfileError](err); ok {
			failField(w, http.StatusUnsupportedMediaType, CodeInvalidRequest, "file", bad.Problem)
			return
		}
		s.failErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.accountInfo(r, nil))
}

// handleRemoveAvatar goes back to initials.
func (s *Server) handleRemoveAvatar(w http.ResponseWriter, r *http.Request) {
	keeper, ok := s.account(w)
	if !ok {
		return
	}
	if err := keeper.RemoveAvatar(); err != nil {
		s.failErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.accountInfo(r, nil))
}

// handleSetCredentials gives an open studio a password, or changes the name
// and password of a guarded one.
func (s *Server) handleSetCredentials(w http.ResponseWriter, r *http.Request) {
	keeper, ok := s.account(w)
	if !ok {
		return
	}
	req, ok := decodeJSON[CredentialsChange](s, w, r)
	if !ok {
		return
	}
	user := strings.TrimSpace(req.User)

	current := s.auth.Account()
	if current == nil {
		// Nothing to confirm against: whoever reaches an open studio can
		// already do everything in it, and a server runs open only where
		// nobody else can reach it.
		if user == "" {
			user = "admin"
		}
		if !checkCredentials(w, user, req.Password, true) {
			return
		}
		account, err := keeper.Create(user, req.Password)
		if err != nil {
			s.failAccount(w, err)
			return
		}
		session, err := s.auth.Start(w, r)
		if err != nil {
			s.failErr(w, err)
			return
		}
		s.log.Info("password set", "user", account.User())
		writeJSON(w, http.StatusOK, s.accountInfo(r, &session))
		return
	}

	if current.Source() != auth.File {
		s.failAccount(w, auth.ErrFixedAccount)
		return
	}
	if user == current.User() {
		user = ""
	}
	if user == "" && req.Password == "" {
		fail(w, http.StatusBadRequest, CodeInvalidRequest,
			"nothing to change: send a new user name, a new password, or both")
		return
	}
	// Checked before the password is confirmed, so that a typo in the new
	// name does not cost one of the attempts guessing is limited to.
	if !checkCredentials(w, user, req.Password, false) {
		return
	}

	confirmed, ok := s.confirm(w, req.CurrentPassword)
	if !ok {
		return
	}
	session, err := s.auth.Session(r)
	if err != nil {
		s.failErr(w, err)
		return
	}
	account, err := keeper.Change(confirmed, user, req.Password)
	if err != nil {
		s.failAccount(w, err)
		return
	}
	renewed, err := s.auth.Renew(w, r, session)
	if err != nil {
		s.failErr(w, err)
		return
	}
	s.log.Info("account changed", "user", account.User())
	writeJSON(w, http.StatusOK, s.accountInfo(r, &renewed))
}

// checkCredentials refuses a name or password that cannot be stored, naming
// the field. An empty one is left alone unless it is required.
func checkCredentials(w http.ResponseWriter, user, password string, required bool) bool {
	if user != "" {
		if err := auth.CheckUser(user); err != nil {
			failField(w, http.StatusBadRequest, CodeInvalidRequest, "user", problemOf(err))
			return false
		}
	}
	if password != "" || required {
		if err := auth.CheckPassword(password); err != nil {
			failField(w, http.StatusBadRequest, CodeInvalidRequest, "password", problemOf(err))
			return false
		}
	}
	return true
}

// problemOf is an auth error in the API's words, without the package prefix.
func problemOf(err error) string {
	return strings.TrimPrefix(err.Error(), "auth: ")
}

// handleRemoveCredentials takes the password away and leaves the studio open.
func (s *Server) handleRemoveCredentials(w http.ResponseWriter, r *http.Request) {
	keeper, ok := s.account(w)
	if !ok {
		return
	}
	req, ok := decodeJSON[PasswordConfirmation](s, w, r)
	if !ok {
		return
	}

	current := s.auth.Account()
	switch {
	case current == nil:
		fail(w, http.StatusBadRequest, CodeInvalidRequest, "this studio has no password to remove")
		return
	case current.Source() != auth.File:
		s.failAccount(w, auth.ErrFixedAccount)
		return
	case !keeper.MayOpen():
		s.failAccount(w, auth.ErrMustStayGuarded)
		return
	}

	confirmed, ok := s.confirm(w, req.CurrentPassword)
	if !ok {
		return
	}
	if err := keeper.Remove(confirmed); err != nil {
		s.failAccount(w, err)
		return
	}
	s.auth.SignOut(w, r)
	s.log.Warn("password removed; the studio is open to this machine")
	writeJSON(w, http.StatusOK, s.accountInfo(r, nil))
}

// handleEndSessions signs out every browser but the caller's.
func (s *Server) handleEndSessions(w http.ResponseWriter, r *http.Request) {
	keeper, ok := s.account(w)
	if !ok {
		return
	}

	current := s.auth.Account()
	if current == nil {
		fail(w, http.StatusBadRequest, CodeInvalidRequest, "this studio has no sessions: it asks for no password")
		return
	}
	session, err := s.auth.Session(r)
	if err != nil {
		s.failErr(w, err)
		return
	}
	if _, err := keeper.Rotate(current); err != nil {
		s.failAccount(w, err)
		return
	}
	renewed, err := s.auth.Renew(w, r, session)
	if err != nil {
		s.failErr(w, err)
		return
	}
	s.log.Info("signed out everywhere else", "user", current.User())
	writeJSON(w, http.StatusOK, s.accountInfo(r, &renewed))
}

// confirm checks the current password before a change that needs it.
//
// A wrong one is 403 rather than 401: the caller is signed in, and a 401
// would tell a client its session had ended, which is not what happened.
func (s *Server) confirm(w http.ResponseWriter, password string) (*auth.Account, bool) {
	account, err := s.auth.Confirm(password)
	if err == nil {
		return account, true
	}
	if errors.Is(err, auth.ErrBadCredentials) {
		failField(w, http.StatusForbidden, CodeWrongPassword, "current_password",
			"the current password is not right")
	} else if !s.failThrottled(w, err) {
		s.failErr(w, err)
	}
	return nil, false
}

// failAccount maps a refusal from the account's keeper onto a response.
func (s *Server) failAccount(w http.ResponseWriter, err error) {
	if bad, ok := errors.AsType[*auth.ProfileError](err); ok {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, bad.Field, bad.Problem)
		return
	}
	switch {
	case errors.Is(err, auth.ErrFixedAccount):
		fail(w, http.StatusConflict, CodeAccountFixed,
			"this account comes from KITE_ADMIN_USER and KITE_ADMIN_PASSWORD, and is changed where they are set")
	case errors.Is(err, auth.ErrMustStayGuarded):
		fail(w, http.StatusConflict, CodePasswordRequired,
			"this server can be reached from other machines, so its studio has to keep a password")
	case errors.Is(err, auth.ErrAccountChanged), errors.Is(err, auth.ErrAlreadyConfigured):
		fail(w, http.StatusConflict, CodeConflict,
			"the account changed while this was being done; load it again and retry")
	default:
		s.failErr(w, err)
	}
}
