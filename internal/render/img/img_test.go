package img

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

// picture is a w by h JPEG, red in its top left quarter and blue elsewhere,
// so a test can tell which way it was turned or which part was kept.
func picture(t *testing.T, w, h int) []byte {
	t.Helper()
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			c := color.RGBA{0, 0, 255, 255}
			if x < w/2 && y < h/2 {
				c = color.RGBA{255, 0, 0, 255}
			}
			m.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, m, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// turned is a JPEG whose EXIF says to show it turned: 6 is a quarter turn
// clockwise, as a phone held upright writes a photo.
func turned(data []byte, orientation byte) []byte {
	tiff := []byte("MM\x00\x2a\x00\x00\x00\x08\x00\x01\x01\x12\x00\x03\x00\x00\x00\x01\x00")
	tiff = append(tiff, orientation, 0, 0, 0, 0, 0, 0)
	segment := append([]byte("Exif\x00\x00"), tiff...)
	app1 := []byte{0xFF, 0xE1, byte((len(segment) + 2) >> 8), byte(len(segment) + 2)}
	out := append([]byte{}, data[:2]...)
	out = append(out, app1...)
	out = append(out, segment...)
	return append(out, data[2:]...)
}

func made(t *testing.T, data []byte, r Recipe) (image.Image, Info) {
	t.Helper()
	out, info, err := produce(data, r)
	if err != nil {
		t.Fatal(err)
	}
	m, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	return m, info
}

func step(t *testing.T, op, spec string) Step {
	t.Helper()
	s, err := ParseStep(op, spec)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Each kind of step makes the size it says: resize keeps the ratio where a
// side is left out, fit only ever scales down, fill crops to the ratio before
// it scales, and crop cuts without scaling.
func TestEachStepMakesTheSizeItSays(t *testing.T) {
	src := picture(t, 400, 200)
	for _, c := range []struct {
		op, spec string
		w, h     int
	}{
		{"resize", "100x", 100, 50},
		{"resize", "x50", 100, 50},
		{"resize", "100x100", 100, 100},
		{"fit", "100x100", 100, 50},
		{"fit", "1000x1000", 400, 200},
		{"fill", "100x100", 100, 100},
		{"crop", "50x80", 50, 80},
	} {
		m, info := made(t, src, Recipe{Steps: []Step{step(t, c.op, c.spec)}})
		if b := m.Bounds(); b.Dx() != c.w || b.Dy() != c.h || info.Width != c.w || info.Height != c.h {
			t.Errorf("%s %s made %v, info %dx%d; want %dx%d", c.op, c.spec, b.Size(), info.Width, info.Height, c.w, c.h)
		}
	}
}

// Fill and crop keep the part of the picture their anchor names.
func TestAnAnchorKeepsItsPart(t *testing.T) {
	src := picture(t, 400, 200) // red in the top left quarter
	red := func(m image.Image) bool {
		r, g, b, _ := m.At(m.Bounds().Dx()/2, m.Bounds().Dy()/2).RGBA()
		return r > 0xc000 && g < 0x4000 && b < 0x4000
	}
	if m, _ := made(t, src, Recipe{Steps: []Step{step(t, "crop", "100x50 topleft")}}); !red(m) {
		t.Error("a crop anchored top left did not keep the red corner")
	}
	if m, _ := made(t, src, Recipe{Steps: []Step{step(t, "fill", "100x100 bottomright")}}); red(m) {
		t.Error("a fill anchored bottom right kept the red corner")
	}
}

// A phone's photo is turned as its EXIF says before anything else is done,
// and what it made says nothing of it, orientation or where it was taken.
func TestAPhotoIsTurnedAsItsEXIFSays(t *testing.T) {
	src := turned(picture(t, 400, 200), 6)
	info, err := Inspect(src)
	if err != nil || info.Width != 200 || info.Height != 400 {
		t.Fatalf("Inspect = %+v, %v; want a 200x400 picture as shown", info, err)
	}
	m, _ := made(t, src, Recipe{Steps: []Step{step(t, "resize", "100x")}})
	if b := m.Bounds(); b.Dx() != 100 || b.Dy() != 200 {
		t.Errorf("made %v, want 100x200 turned upright", b.Size())
	}
	// A quarter turn clockwise puts the top left corner at the top right.
	r, _, _, _ := m.At(75, 25).RGBA()
	if r < 0xc000 {
		t.Error("the red corner is not at the top right after the turn")
	}
	out, _, _ := produce(src, Recipe{})
	if bytes.Contains(out, []byte("Exif")) {
		t.Error("the made picture carries the source's EXIF")
	}
}

// A picture is written in the format asked for, a transparent one on white
// when that is a JPEG, and a WebP, which cannot be written, as a JPEG.
func TestAPictureIsWrittenAsAsked(t *testing.T) {
	src := picture(t, 40, 20)
	_, info := made(t, src, Recipe{Format: "png"})
	if info.Format != "png" {
		t.Errorf("format = %s, want png", info.Format)
	}

	clear := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	if err := png.Encode(&buf, clear); err != nil {
		t.Fatal(err)
	}
	m, info := made(t, buf.Bytes(), Recipe{Format: "jpeg"})
	if r, g, b, _ := m.At(1, 1).RGBA(); info.Format != "jpeg" || r < 0xf000 || g < 0xf000 || b < 0xf000 {
		t.Errorf("a transparent picture written as %s is %v, want white", info.Format, m.At(1, 1))
	}

	// The 1x1 lossy WebP browsers are tested for support with.
	webp, _ := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	if info, err := Inspect(webp); err != nil || info.Format != "webp" {
		t.Fatalf("Inspect(webp) = %+v, %v", info, err)
	}
	_, info = made(t, webp, Recipe{Steps: []Step{step(t, "resize", "4x")}})
	if info.Format != "jpeg" || info.Width != 4 {
		t.Errorf("a WebP was made into %+v, want a 4x4 JPEG", info)
	}
}

// A size or an option a step cannot use is refused as it is written, and
// saying what would do instead.
func TestAStepThatCannotBeMadeIsRefused(t *testing.T) {
	for _, c := range []struct{ op, spec, want string }{
		{"resize", "", "a size is needed"},
		{"resize", "big", "not a size"},
		{"resize", "x", "a width or a height"},
		{"fit", "800x", "both a width and a height"},
		{"fill", "800x600 middle", "not an anchor"},
		{"resize", "800x600 top", "not something resize takes"},
		{"resize", "99999x", "not a size"},
	} {
		if _, err := ParseStep(c.op, c.spec); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s %q: error = %v, want one containing %q", c.op, c.spec, err, c.want)
		}
	}
	if _, err := ParseFormat("webp"); err == nil || !strings.Contains(err.Error(), "cannot write webp yet") {
		t.Errorf("webp: %v", err)
	}
}

// A picture is made once for a source and a recipe: asked again, in this run
// or the next one, it is found rather than made, and a published name leads
// back to it.
func TestAPictureIsMadeOnce(t *testing.T) {
	dir := t.TempDir()
	src := picture(t, 400, 200)
	reads := 0
	read := func() ([]byte, error) { reads++; return src, nil }
	r := Recipe{Steps: []Step{step(t, "resize", "100x")}}

	first, err := NewProcessor(dir).Make("hash-of-src", read, r)
	if err != nil {
		t.Fatal(err)
	}
	again, err := NewProcessor(dir).Make("hash-of-src", read, r)
	if err != nil {
		t.Fatal(err)
	}
	if reads != 1 || again.Key != first.Key || again.Width != 100 {
		t.Errorf("read %d times, keys %s and %s; want one read and one picture", reads, first.Key, again.Key)
	}
	if other := Key("hash-of-src", r.With(step(t, "crop", "10x10"))); other == first.Key {
		t.Error("another recipe has the same key")
	}

	name := PublishedName("river", first)
	found, ok := NewProcessor(dir).Lookup("posts/trip/" + name)
	if !ok || found.Key != first.Key {
		t.Fatalf("Lookup(%s) = %v, %v", name, found, ok)
	}
	want, _ := first.Bytes()
	if got, _ := found.Bytes(); !bytes.Equal(got, want) {
		t.Error("the picture found is not the one made")
	}
	if _, ok := NewProcessor(dir).Lookup("posts/trip/river.jpg"); ok {
		t.Error("a source's own name was taken for a made picture")
	}
}
