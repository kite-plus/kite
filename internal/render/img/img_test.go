package img

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

// picture is a w by h JPEG, red in its top left quarter and blue elsewhere,
// so a test can tell which way it was turned or which part was kept.
func picture(t testing.TB, w, h int) []byte {
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

// A picture is written in the format asked for, and as its source is
// otherwise: a transparent one on white when it is written as a JPEG, which
// holds no transparency, and kept transparent in a WebP.
func TestAPictureIsWrittenAsAsked(t *testing.T) {
	src := picture(t, 40, 20)
	for _, format := range []string{"png", "gif", "webp"} {
		m, info := made(t, src, Recipe{Format: format})
		if info.Format != format || m.Bounds().Dx() != 40 {
			t.Errorf("written as %s: %+v, %v", format, info, m.Bounds())
		}
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
	m, info = made(t, buf.Bytes(), Recipe{Format: "webp"})
	if _, _, _, a := m.At(1, 1).RGBA(); info.Format != "webp" || a != 0 {
		t.Errorf("a transparent picture written as %s is %v, want transparent", info.Format, m.At(1, 1))
	}

	// The 1x1 lossy WebP browsers are tested for support with.
	webp, _ := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	if info, err := Inspect(webp); err != nil || info.Format != "webp" {
		t.Fatalf("Inspect(webp) = %+v, %v", info, err)
	}
	_, info = made(t, webp, Recipe{Steps: []Step{step(t, "resize", "4x")}})
	if info.Format != "webp" || info.Width != 4 {
		t.Errorf("a WebP was made into %+v, want a 4x4 WebP", info)
	}
}

// A WebP keeps its black black and its white white when it is made into
// another picture. Its colors are stored in the range video uses, 16 for
// black and 235 for white, and read as JPEG's they came out grey.
func TestAWebPKeepsItsBlackAndWhite(t *testing.T) {
	m := image.NewRGBA(image.Rect(0, 0, 64, 32))
	for y := range 32 {
		for x := range 64 {
			c := color.RGBA{0, 0, 0, 255}
			if x >= 32 {
				c = color.RGBA{255, 255, 255, 255}
			}
			m.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, m); err != nil {
		t.Fatal(err)
	}
	webp, _, err := produce(buf.Bytes(), Recipe{Format: "webp", Quality: 90})
	if err != nil {
		t.Fatal(err)
	}
	back, _ := made(t, webp, Recipe{Format: "png"})
	black, _, _, _ := back.At(8, 16).RGBA()
	white, _, _, _ := back.At(56, 16).RGBA()
	if black>>8 > 4 || white>>8 < 251 {
		t.Errorf("black reads as %d and white as %d, want 0 and 255", black>>8, white>>8)
	}
}

// The same source and recipe make the same bytes on every machine, which is
// what lets a site built on a laptop and on a CI runner publish the same
// files, and a cache made on one be trusted on another. CI runs this on
// amd64 and arm64, whose floating point differs where a compiler fuses a
// multiplication and an addition; a change here means a machine now makes
// other bytes, or the code that makes pictures changed and its version with
// it.
func TestAPictureIsTheSameOnEveryMachine(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 160, 120))
	seed := uint32(7)
	for y := range 120 {
		for x := range 160 {
			seed = seed*1664525 + 1013904223
			noise := uint8(seed >> 28)
			c := color.NRGBA{uint8(x + int(noise)), uint8(y*2 + int(noise)), uint8((x+y)/2 + int(noise)), 255}
			if (x-50)*(x-50)+(y-40)*(y-40) < 400 {
				c = color.NRGBA{230, 60, 40, 255}
			}
			if x > 140 {
				c.A = uint8(y * 2)
			}
			src.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	photo := turned(picture(t, 120, 80), 6)
	for _, c := range []struct {
		name   string
		source []byte
		recipe Recipe
		want   string
	}{
		{"resize to jpeg", buf.Bytes(), Recipe{Steps: []Step{step(t, "resize", "80x")}, Format: "jpeg", Quality: 80}, "5e01c3417c1c018f"},
		{"fill to png", buf.Bytes(), Recipe{Steps: []Step{step(t, "fill", "64x64 topright")}}, "2bb66b381f53bb40"},
		{"fit to webp", buf.Bytes(), Recipe{Steps: []Step{step(t, "fit", "100x100")}, Format: "webp"}, "c802f674051ab457"},
		{"crop to gif", buf.Bytes(), Recipe{Steps: []Step{step(t, "crop", "50x40")}, Format: "gif"}, "cba15e309f6b1c3e"},
		{"turned photo", photo, Recipe{Steps: []Step{step(t, "resize", "x60")}, Format: "webp", Quality: 60}, "20404a3c8835980d"},
	} {
		out, _, err := produce(c.source, c.recipe)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(out)
		if got := hex.EncodeToString(sum[:8]); got != c.want {
			t.Errorf("%s made %s, want %s", c.name, got, c.want)
		}
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
	if _, err := ParseFormat("avif"); err == nil || !strings.Contains(err.Error(), "cannot write avif") {
		t.Errorf("avif: %v", err)
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
