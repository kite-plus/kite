package img

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // reads WebP, which a phone or an export may give
)

// maxPixels bounds what is decoded: a small file can hold a picture that
// takes gigabytes to decode.
const maxPixels = 100_000_000

// Info is what a picture's header says of it, turned as it is shown.
type Info struct {
	Format        string
	Width, Height int
}

// Inspect reads a picture's format and size without decoding it. A JPEG
// whose EXIF says it is turned on its side is as wide as it is shown.
func Inspect(data []byte) (Info, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Info{}, fmt.Errorf("img: not a picture Kite can read: %w", err)
	}
	info := Info{Format: format, Width: cfg.Width, Height: cfg.Height}
	if format == "jpeg" && orientation(data) >= 5 {
		info.Width, info.Height = info.Height, info.Width
	}
	return info, nil
}

// produce carries out a recipe on a picture's bytes, and returns what it
// wrote and in which format. What the source's metadata says, such as where
// a photo was taken, is not carried over.
func produce(data []byte, r Recipe) ([]byte, Info, error) {
	info, err := Inspect(data)
	if err != nil {
		return nil, Info{}, err
	}
	if info.Width*info.Height > maxPixels {
		return nil, Info{}, fmt.Errorf("img: a %dx%d picture is larger than Kite decodes", info.Width, info.Height)
	}
	m, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, Info{}, fmt.Errorf("img: decode: %w", err)
	}
	if format == "jpeg" {
		m = orient(m, orientation(data))
	}
	for _, s := range r.Steps {
		m = apply(m, s)
	}

	out := r.Format
	if out == "" {
		out = format
	}
	if out == "webp" {
		out = "jpeg"
		if !opaque(m) {
			out = "png"
		}
	}
	var buf bytes.Buffer
	switch out {
	case "jpeg":
		q := r.Quality
		if q <= 0 {
			q = DefaultQuality
		}
		err = jpeg.Encode(&buf, flatten(m), &jpeg.Options{Quality: min(q, 100)})
	case "png":
		err = png.Encode(&buf, m)
	case "gif":
		err = gif.Encode(&buf, m, nil)
	default:
		err = fmt.Errorf("img: cannot write %s", out)
	}
	if err != nil {
		return nil, Info{}, fmt.Errorf("img: encode: %w", err)
	}
	b := m.Bounds()
	return buf.Bytes(), Info{Format: out, Width: b.Dx(), Height: b.Dy()}, nil
}

// apply carries out one step.
func apply(m image.Image, s Step) image.Image {
	b := m.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	switch s.Op {
	case "resize":
		tw, th := s.Width, s.Height
		if tw == 0 {
			tw = whole(w * float64(th) / h)
		}
		if th == 0 {
			th = whole(h * float64(tw) / w)
		}
		return scale(m, tw, th)
	case "fit":
		if w <= float64(s.Width) && h <= float64(s.Height) {
			return m
		}
		ratio := math.Min(float64(s.Width)/w, float64(s.Height)/h)
		return scale(m, whole(w*ratio), whole(h*ratio))
	case "fill":
		// The largest box of the size's ratio the picture holds, scaled.
		cw, ch := w, w*float64(s.Height)/float64(s.Width)
		if ch > h {
			cw, ch = h*float64(s.Width)/float64(s.Height), h
		}
		return scale(crop(m, whole(cw), whole(ch), s.Anchor), s.Width, s.Height)
	case "crop":
		return crop(m, min(s.Width, b.Dx()), min(s.Height, b.Dy()), s.Anchor)
	}
	return m
}

func whole(f float64) int { return max(1, int(math.Round(f))) }

// scale draws a picture at another size with Catmull-Rom, which keeps a
// photo sharp where a cheaper filter blurs it.
func scale(m image.Image, w, h int) image.Image {
	if b := m.Bounds(); b.Dx() == w && b.Dy() == h {
		return m
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), m, m.Bounds(), draw.Src, nil)
	return dst
}

// crop cuts a w by h box out of a picture at an anchor.
func crop(m image.Image, w, h int, anchor string) image.Image {
	b := m.Bounds()
	x := b.Min.X + (b.Dx()-w)/2
	y := b.Min.Y + (b.Dy()-h)/2
	switch anchor {
	case "left", "topleft", "bottomleft":
		x = b.Min.X
	case "right", "topright", "bottomright":
		x = b.Max.X - w
	}
	switch anchor {
	case "top", "topleft", "topright":
		y = b.Min.Y
	case "bottom", "bottomleft", "bottomright":
		y = b.Max.Y - h
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), m, image.Pt(x, y), draw.Src)
	return dst
}

// opaque reports whether a picture has no transparent pixel.
func opaque(m image.Image) bool {
	if o, ok := m.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	return false
}

// flatten puts a picture that has transparent pixels on white, since a JPEG
// holds none and would show them black.
func flatten(m image.Image) image.Image {
	if opaque(m) {
		return m
	}
	b := m.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), m, b.Min, draw.Over)
	return dst
}

// orientation is the EXIF orientation a JPEG gives, from 1 to 8, and 1 when
// it gives none. A phone keeps a photo as the sensor saw it and says here how
// to turn it.
func orientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xFF { // fill byte
			i++
			continue
		}
		if marker == 0xD9 || marker == 0xDA { // the end, or the picture itself
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2:]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		if segment := data[i+4 : i+2+size]; marker == 0xE1 && bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
			return exifOrientation(segment[6:])
		}
		i += 2 + size
	}
	return 1
}

// exifOrientation reads tag 0x0112 of the first IFD of a TIFF header.
func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	ifd := int(order.Uint32(tiff[4:]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 1
	}
	for k := range int(order.Uint16(tiff[ifd:])) {
		e := ifd + 2 + 12*k
		if e+12 > len(tiff) {
			return 1
		}
		if order.Uint16(tiff[e:]) == 0x0112 {
			if v := int(order.Uint16(tiff[e+8:])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// orient turns a picture as its EXIF orientation says it is shown.
func orient(m image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return m
	}
	b := m.Bounds()
	w, h := b.Dx(), b.Dy()
	src := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(src, src.Bounds(), m, b.Min, draw.Src)
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := range h {
		for x := range w {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			copy(dst.Pix[dy*dst.Stride+dx*4:dy*dst.Stride+dx*4+4], src.Pix[y*src.Stride+x*4:y*src.Stride+x*4+4])
		}
	}
	return dst
}
