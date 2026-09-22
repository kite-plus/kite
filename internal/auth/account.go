// Package auth guards the admin with a single local account.
//
// V1 is deliberately one account. A self-hosted site has one operator, and
// what would make several of them mean anything -- roles, invitations,
// per-author audit -- belongs with the database store rather than with a
// project made of files. What cannot wait is the part that is painful to
// retrofit: credentials stored safely, a session that cannot be forged, and a
// server that refuses to put an unprotected admin on a public address.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// File is where an account is stored, relative to the project root.
//
// It lives under .kite because that directory is already ignored by every
// project kite creates, and a password hash that reaches a public repository
// is the one mistake this file must make impossible. The rest of .kite is
// derived and rebuildable; this is not, so a deployment has to keep it --
// which is also why the environment can supply the account instead.
const File = ".kite/secrets/account.json"

// ErrNoAccount reports a project with no account configured. It is not a
// failure on its own: a local preview with no account is simply open.
var ErrNoAccount = errors.New("auth: no account configured")

// ErrBadCredentials is returned for a wrong user or a wrong password, which
// are deliberately indistinguishable to a caller.
var ErrBadCredentials = errors.New("auth: incorrect user name or password")

// ErrAlreadyConfigured reports an attempt to create an account where one
// already exists, which is how first-run setup refuses to run twice.
var ErrAlreadyConfigured = errors.New("auth: this project already has an account")

// MinPasswordLength is the shortest password this will store. Length is the
// only property worth enforcing: composition rules push people towards
// "Passw0rd!" and away from a passphrase.
const MinPasswordLength = 8

// Argon2id parameters, chosen so a single verification costs roughly a tenth
// of a second on a small server. They travel inside every stored hash, so
// raising them later is a change to new passwords rather than a migration.
const (
	hashTime    = 3
	hashMemory  = 64 * 1024 // KiB
	hashThreads = 2
	hashLength  = 32
	saltLength  = 16
)

// Account is one operator's credentials.
type Account struct {
	user     string
	password string // PHC encoded argon2id hash
	secret   []byte // signs this account's sessions

	// source says where the account came from, which the CLI reports and a
	// deployment needs to know: an account from the environment cannot be
	// changed through a running server.
	source string
}

// User is the name this account signs in with.
func (a *Account) User() string { return a.user }

// Source describes where the account was read from.
func (a *Account) Source() string { return a.source }

// stored is the on-disk shape, versioned so a later format can be recognized
// rather than guessed at.
type stored struct {
	Version   int       `json:"version"`
	User      string    `json:"user"`
	Password  string    `json:"password"`
	Secret    string    `json:"secret"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Open returns the account guarding a project.
//
// The environment wins over the file. A container sets KITE_ADMIN_PASSWORD
// deliberately and has no way to run a command inside the image first, so an
// account left behind in a mounted volume must not silently take precedence
// over the one the operator just configured.
func Open(root string) (*Account, error) {
	if a, err := FromEnv(); err == nil {
		return a, nil
	} else if !errors.Is(err, ErrNoAccount) {
		return nil, err
	}
	return Load(root)
}

// FromEnv builds an account from KITE_ADMIN_USER and KITE_ADMIN_PASSWORD, or
// KITE_ADMIN_PASSWORD_HASH for a deployment that would rather not put a
// plaintext password in a process listing.
func FromEnv() (*Account, error) {
	user := strings.TrimSpace(os.Getenv("KITE_ADMIN_USER"))
	password := os.Getenv("KITE_ADMIN_PASSWORD")
	hash := os.Getenv("KITE_ADMIN_PASSWORD_HASH")

	if password == "" && hash == "" {
		return nil, ErrNoAccount
	}
	if user == "" {
		user = "admin"
	}

	if hash == "" {
		// Hashing a password the operator already has in plaintext buys
		// nothing against an attacker who can read the environment. It is
		// done anyway so that everything past this point has one shape, and
		// so a crash dump or a log of this process holds a hash rather than
		// the password itself.
		var err error
		if hash, err = Hash(password); err != nil {
			return nil, err
		}
	}
	if _, err := parsePHC(hash); err != nil {
		return nil, fmt.Errorf("auth: KITE_ADMIN_PASSWORD_HASH: %w", err)
	}

	return &Account{
		user:     user,
		password: hash,
		// No stored secret to sign with, so one is derived from the hash.
		// Sessions then survive a restart, and changing the password ends
		// them, which is the same behavior a file-backed account has.
		secret: derive([]byte(hash), "env secret"),
		source: "environment",
	}, nil
}

// Load reads the account stored in a project, or reports [ErrNoAccount].
func Load(root string) (*Account, error) {
	path := filepath.Join(root, filepath.FromSlash(File))

	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, ErrNoAccount
	case err != nil:
		return nil, err
	}

	var s stored
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("auth: parse %s: %w", File, err)
	}
	if s.Version != 1 {
		return nil, fmt.Errorf("auth: %s was written by a later version of kite", File)
	}
	if _, err := parsePHC(s.Password); err != nil {
		return nil, fmt.Errorf("auth: %s: %w", File, err)
	}
	secret, err := base64.RawStdEncoding.DecodeString(s.Secret)
	if err != nil || len(secret) < 32 {
		return nil, fmt.Errorf("auth: %s: the session secret is missing or too short", File)
	}

	return &Account{user: s.User, password: s.Password, secret: secret, source: File}, nil
}

// Configured reports whether signing in is possible at all, which is what
// decides whether a server may be reachable from outside this machine.
func Configured(root string) bool {
	_, err := Open(root)
	return err == nil
}

// SetPassword creates or replaces the stored account.
//
// The session secret is kept across a change so that the file keeps one
// identity, and every existing session still ends, because a session is
// signed with a key derived from the secret and the password together.
func SetPassword(root, user, password string) (*Account, error) {
	if strings.TrimSpace(user) == "" {
		return nil, errors.New("auth: a user name is required")
	}
	if len([]rune(password)) < MinPasswordLength {
		return nil, fmt.Errorf("auth: a password needs at least %d characters", MinPasswordLength)
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	if existing, err := Load(root); err == nil {
		secret = existing.secret
	} else if !errors.Is(err, ErrNoAccount) {
		return nil, err
	}

	hash, err := Hash(password)
	if err != nil {
		return nil, err
	}
	account := &Account{user: user, password: hash, secret: secret, source: File}

	body, err := json.MarshalIndent(stored{
		Version:   1,
		User:      user,
		Password:  hash,
		Secret:    base64.RawStdEncoding.EncodeToString(secret),
		UpdatedAt: time.Now().UTC(),
	}, "", "  ")
	if err != nil {
		return nil, err
	}

	path := filepath.Join(root, filepath.FromSlash(File))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	// Written through a temporary file so that an interrupted write cannot
	// leave a project with an account nobody can sign in to.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(body, '\n'), 0o600); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	return account, nil
}

// Remove deletes the stored account, leaving the admin open on a loopback
// address and refused on any other.
func Remove(root string) error {
	err := os.Remove(filepath.Join(root, filepath.FromSlash(File)))
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNoAccount
	}
	return err
}

// Verify reports whether a name and password match this account.
//
// Both comparisons are constant time, and a wrong name still pays for a hash,
// so that response time does not answer "is there a user called admin".
func (a *Account) Verify(user, password string) error {
	params, err := parsePHC(a.password)
	if err != nil {
		return err
	}
	sum := argon2.IDKey([]byte(password), params.salt,
		params.time, params.memory, params.threads, uint32(len(params.hash)))

	ok := subtle.ConstantTimeCompare(sum, params.hash)
	ok &= subtle.ConstantTimeCompare([]byte(user), []byte(a.user))
	if ok != 1 {
		return ErrBadCredentials
	}
	return nil
}

// Hash encodes a password as an argon2id PHC string.
func Hash(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	sum := argon2.IDKey([]byte(password), salt, hashTime, hashMemory, hashThreads, hashLength)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, hashMemory, hashTime, hashThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum)), nil
}

// phc is a decoded hash string, carrying the cost it was produced with.
type phc struct {
	time, memory uint32
	threads      uint8
	salt, hash   []byte
}

func parsePHC(s string) (phc, error) {
	var p phc
	parts := strings.Split(s, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return p, errors.New("not an argon2id hash")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return p, errors.New("unsupported argon2 version")
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return p, errors.New("unreadable argon2 parameters")
	}

	var err error
	if p.salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		return p, errors.New("unreadable salt")
	}
	if p.hash, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil {
		return p, errors.New("unreadable hash")
	}
	if len(p.salt) == 0 || len(p.hash) == 0 {
		return p, errors.New("empty salt or hash")
	}
	return p, nil
}

// sessionKey signs this account's sessions.
//
// It binds the stored secret to the password hash, so changing the password
// signs out every browser without needing a list of sessions to revoke --
// which a server that keeps nothing in memory could not have.
func (a *Account) sessionKey() []byte {
	return derive(a.secret, "session key:"+a.user+":"+a.password)
}

func derive(key []byte, label string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("kite auth v1/" + label))
	return mac.Sum(nil)
}
