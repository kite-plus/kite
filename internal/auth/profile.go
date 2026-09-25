package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// ProfileFile holds how the person who runs the studio is shown in it.
//
// It sits beside the account because it has to survive a deployment the same
// way and has no business in the repository, and apart from it because a
// studio with no password is still used by somebody, and one whose password
// comes from the environment has no file of its own to write to.
const ProfileFile = ".kite/secrets/profile.json"

// AvatarFile holds the picture shown beside the name.
const AvatarFile = ".kite/secrets/avatar"

// MaxAvatarSize bounds a stored picture. The studio shrinks one before
// sending it, so this leaves room for a picture sent some other way rather
// than being the size of an ordinary one.
const MaxAvatarSize = 1 << 20

// MaxNameLength bounds the name shown in the studio's narrowest places.
const MaxNameLength = 64

// maxEmailLength is the longest address SMTP can carry.
const maxEmailLength = 254

// avatarTypes are the pictures a browser shows in an img element without
// running anything. SVG is not one of them.
var avatarTypes = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true,
}

// ErrNoAvatar reports a studio with no picture chosen.
var ErrNoAvatar = errors.New("auth: no avatar has been chosen")

// Profile is how the person using the studio is shown in it.
//
// It is never published. The site's author is site.author in kite.yaml, which
// a build can read; this is kept out of the repository, so a build on another
// machine could not, and two builds of one commit would differ.
type Profile struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// ProfileError is a profile value that cannot be stored.
type ProfileError struct {
	Field   string
	Problem string
}

func (e *ProfileError) Error() string { return "auth: " + e.Field + ": " + e.Problem }

type storedProfile struct {
	Version   int       `json:"version"`
	Name      string    `json:"name,omitempty"`
	Email     string    `json:"email,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoadProfile reads the profile stored in a project. A project that never
// stored one has an empty profile, which is not an error.
func LoadProfile(root string) (Profile, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ProfileFile)))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return Profile{}, nil
	case err != nil:
		return Profile{}, err
	}

	var s storedProfile
	if err := json.Unmarshal(data, &s); err != nil {
		return Profile{}, fmt.Errorf("auth: parse %s: %w", ProfileFile, err)
	}
	if s.Version != 1 {
		return Profile{}, fmt.Errorf("auth: %s was written by a later version of kite", ProfileFile)
	}
	return Profile{Name: s.Name, Email: s.Email}, nil
}

// SaveProfile stores a profile, trimmed, and returns it as stored.
func SaveProfile(root string, p Profile) (Profile, error) {
	p.Name = strings.TrimSpace(p.Name)
	p.Email = strings.TrimSpace(p.Email)

	switch {
	case len([]rune(p.Name)) > MaxNameLength:
		return Profile{}, &ProfileError{"name", fmt.Sprintf("a name can have at most %d characters", MaxNameLength)}
	case strings.ContainsFunc(p.Name, unicode.IsControl):
		return Profile{}, &ProfileError{"name", "a name cannot hold control characters"}
	case p.Email != "" && !wellFormedEmail(p.Email):
		return Profile{}, &ProfileError{"email", "this is not an email address"}
	}

	body, err := json.MarshalIndent(storedProfile{
		Version:   1,
		Name:      p.Name,
		Email:     p.Email,
		UpdatedAt: time.Now().UTC(),
	}, "", "  ")
	if err != nil {
		return Profile{}, err
	}
	return p, writePrivate(root, ProfileFile, append(body, '\n'))
}

// wellFormedEmail accepts a bare address and nothing around it: a display
// name would be a second name, and the profile already has one.
func wellFormedEmail(s string) bool {
	if len(s) > maxEmailLength {
		return false
	}
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s && addr.Name == ""
}

// LoadAvatar reads the stored picture and the media type it was stored as.
func LoadAvatar(root string) ([]byte, string, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(AvatarFile)))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, "", ErrNoAvatar
	case err != nil:
		return nil, "", err
	}
	kind := http.DetectContentType(data)
	if !avatarTypes[kind] {
		// Only this package writes the file, and it checked the type. What
		// is there now was put there by something else, so it is not served
		// under a type it was never checked against.
		return nil, "", fmt.Errorf("auth: %s is not a picture this can show", AvatarFile)
	}
	return data, kind, nil
}

// SaveAvatar stores a picture and returns its media type.
//
// The type is read from the bytes rather than taken from a file name or a
// header, since it is what the picture will be served as.
func SaveAvatar(root string, data []byte) (string, error) {
	if len(data) > MaxAvatarSize {
		return "", &ProfileError{"avatar", fmt.Sprintf("a picture can be at most %d KB", MaxAvatarSize>>10)}
	}
	kind := http.DetectContentType(data)
	if !avatarTypes[kind] {
		return "", &ProfileError{"avatar", "a picture has to be PNG, JPEG, GIF or WebP"}
	}
	return kind, writePrivate(root, AvatarFile, data)
}

// RemoveAvatar deletes the stored picture. Removing one that is not there
// is not an error: the outcome is the one that was asked for.
func RemoveAvatar(root string) error {
	err := os.Remove(filepath.Join(root, filepath.FromSlash(AvatarFile)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
