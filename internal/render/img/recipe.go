// Package img makes the pictures a theme asks for from the ones a page bundle
// holds, smaller or cropped or in another format, and keeps each one it made,
// so that no picture is made twice and a build and a server hand out the same
// bytes.
package img

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Step is one thing done to a picture's size.
type Step struct {
	// Op is resize, which scales to the size given and keeps the ratio when
	// one side is left out; fit, which scales down to fit inside the size;
	// fill, which crops to the size's ratio and scales to it; or crop, which
	// cuts the size out without scaling.
	Op string

	// Width and Height are in pixels; 0 is a side resize leaves to the ratio.
	Width, Height int

	// Anchor is the part of the picture fill and crop keep: center, top,
	// bottom, left, right, topleft, topright, bottomleft or bottomright.
	Anchor string
}

// Recipe is what a theme asks of one picture: the steps in order, and what
// the result is written as.
type Recipe struct {
	Steps []Step

	// Format is jpeg, png, gif or webp; empty keeps the source's.
	Format string

	// Quality is a JPEG's or a WebP's, from 1 to 100; 0 is DefaultQuality.
	Quality int
}

// DefaultQuality is the quality a JPEG or a WebP is written at when none is
// asked for.
const DefaultQuality = 75

var anchors = []string{"center", "top", "bottom", "left", "right", "topleft", "topright", "bottomleft", "bottomright"}

// ParseStep reads a step as a theme writes it: a size such as 800x, x600 or
// 800x600, and for fill and crop an anchor after it, as "600x400 top".
func ParseStep(op, spec string) (Step, error) {
	s := Step{Op: op, Anchor: "center"}
	fields := strings.Fields(strings.ToLower(spec))
	if len(fields) == 0 {
		return s, fmt.Errorf("img.%s: a size is needed, such as 800x600", Title(op))
	}
	w, h, ok := strings.Cut(fields[0], "x")
	if !ok {
		return s, fmt.Errorf("img.%s: %q is not a size such as 800x600", Title(op), fields[0])
	}
	var err error
	if s.Width, err = side(w); err == nil {
		s.Height, err = side(h)
	}
	switch {
	case err != nil:
		return s, fmt.Errorf("img.%s: %q is not a size such as 800x600", Title(op), fields[0])
	case s.Width == 0 && s.Height == 0:
		return s, fmt.Errorf("img.%s: a size needs a width or a height", Title(op))
	case op != "resize" && (s.Width == 0 || s.Height == 0):
		return s, fmt.Errorf("img.%s: %s needs both a width and a height, such as 800x600", Title(op), op)
	}
	for _, f := range fields[1:] {
		if op != "fill" && op != "crop" {
			return s, fmt.Errorf("img.%s: %q is not something %s takes", Title(op), f, op)
		}
		if !slices.Contains(anchors, f) {
			return s, fmt.Errorf("img.%s: %q is not an anchor (want %s)", Title(op), f, strings.Join(anchors, ", "))
		}
		s.Anchor = f
	}
	if op == "resize" || op == "fit" {
		s.Anchor = ""
	}
	return s, nil
}

// side reads one side of a size: empty for none, or a number of pixels up to
// a size no page shows a picture at.
func side(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 || n > 16384 {
		return 0, fmt.Errorf("bad side %q", s)
	}
	return n, nil
}

// ParseFormat reads what a picture is written as.
func ParseFormat(format string) (string, error) {
	switch f := strings.ToLower(strings.TrimPrefix(format, ".")); f {
	case "jpeg", "jpg":
		return "jpeg", nil
	case "png", "gif", "webp":
		return f, nil
	case "avif":
		return "", fmt.Errorf("img.Format: Kite cannot write avif; write webp, jpeg or png")
	default:
		return "", fmt.Errorf("img.Format: %q is not a format (want jpeg, png, gif or webp)", format)
	}
}

// With returns the recipe with one more step.
func (r Recipe) With(s Step) Recipe {
	r.Steps = append(append([]Step(nil), r.Steps...), s)
	return r
}

// String writes the recipe out in one canonical way, which is what names the
// pictures it makes.
func (r Recipe) String() string {
	var b strings.Builder
	for _, s := range r.Steps {
		fmt.Fprintf(&b, "%s %dx%d %s;", s.Op, s.Width, s.Height, s.Anchor)
	}
	fmt.Fprintf(&b, "format %s;quality %d", r.Format, r.Quality)
	return b.String()
}

// Title is an operation as its img function is named, as Resize.
func Title(op string) string {
	if op == "" {
		return op
	}
	return strings.ToUpper(op[:1]) + op[1:]
}
