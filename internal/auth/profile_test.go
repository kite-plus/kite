package auth_test

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/auth"
)

func TestAProfileIsTrimmedCheckedAndReadBack(t *testing.T) {
	root := t.TempDir()

	empty, err := auth.LoadProfile(root)
	if err != nil || empty != (auth.Profile{}) {
		t.Fatalf("LoadProfile before any was saved = %+v, %v; want empty and no error", empty, err)
	}

	saved, err := auth.SaveProfile(root, auth.Profile{Name: "  Ada Lovelace ", Email: " ada@example.com "})
	if err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	want := auth.Profile{Name: "Ada Lovelace", Email: "ada@example.com"}
	if saved != want {
		t.Errorf("saved = %+v, want %+v", saved, want)
	}
	if got, err := auth.LoadProfile(root); err != nil || got != want {
		t.Errorf("LoadProfile = %+v, %v; want %+v", got, err, want)
	}

	for _, c := range []struct {
		profile auth.Profile
		field   string
	}{
		{auth.Profile{Name: strings.Repeat("名", auth.MaxNameLength+1)}, "name"},
		{auth.Profile{Name: "Ada\x00"}, "name"},
		{auth.Profile{Email: "ada"}, "email"},
		{auth.Profile{Email: "Ada <ada@example.com>"}, "email"},
		{auth.Profile{Email: "ada@example.com, bob@example.com"}, "email"},
	} {
		_, err := auth.SaveProfile(root, c.profile)
		bad, ok := errors.AsType[*auth.ProfileError](err)
		if !ok || bad.Field != c.field {
			t.Errorf("SaveProfile(%+v) = %v, want a problem with %s", c.profile, err, c.field)
		}
	}
	// A refused profile leaves the stored one as it was.
	if got, _ := auth.LoadProfile(root); got != want {
		t.Errorf("after refusals the profile is %+v, want %+v", got, want)
	}

	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(auth.ProfileFile)))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}

func picture(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// The picture is served from the studio's own origin, so what is kept is
// only what a browser will show without running anything.
func TestOnlyAPictureABrowserShowsSafelyIsKeptAsTheAvatar(t *testing.T) {
	root := t.TempDir()

	if _, _, err := auth.LoadAvatar(root); !errors.Is(err, auth.ErrNoAvatar) {
		t.Fatalf("LoadAvatar before any was saved = %v, want ErrNoAvatar", err)
	}

	pic := picture(t)
	kind, err := auth.SaveAvatar(root, pic)
	if err != nil || kind != "image/png" {
		t.Fatalf("SaveAvatar(png) = %q, %v; want image/png", kind, err)
	}
	data, kind, err := auth.LoadAvatar(root)
	if err != nil || kind != "image/png" || !bytes.Equal(data, pic) {
		t.Fatalf("LoadAvatar = %d bytes, %q, %v; want the png back", len(data), kind, err)
	}

	for name, data := range map[string][]byte{
		"svg":      []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"html":     []byte("<!doctype html><title>x</title>"),
		"too big":  append(append([]byte{}, pic...), make([]byte, auth.MaxAvatarSize)...),
		"nothing":  nil,
		"text":     []byte("just some text"),
		"svg+png?": append([]byte("<svg>"), pic...),
	} {
		if _, err := auth.SaveAvatar(root, data); err == nil {
			t.Errorf("SaveAvatar(%s) was accepted", name)
		}
	}
	// A refused picture leaves the one before it in place.
	if got, _, _ := auth.LoadAvatar(root); !bytes.Equal(got, pic) {
		t.Error("a refused picture replaced the stored one")
	}

	if err := auth.RemoveAvatar(root); err != nil {
		t.Fatalf("RemoveAvatar: %v", err)
	}
	if _, _, err := auth.LoadAvatar(root); !errors.Is(err, auth.ErrNoAvatar) {
		t.Errorf("LoadAvatar after removal = %v, want ErrNoAvatar", err)
	}
	if err := auth.RemoveAvatar(root); err != nil {
		t.Errorf("removing a picture that is not there = %v, want nil", err)
	}
}
