package img

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"

	"github.com/deepteams/webp"
)

// latitude is the value a phone writes for 31°13'45.12", which must not be
// left anywhere in a picture once its place is taken out.
var latitude = []byte{0, 0, 0, 31, 0, 0, 0, 1, 0, 0, 0, 13, 0, 0, 0, 1, 0, 0, 0x11, 0xA0, 0, 0, 0, 100}

// tiffWithPlace is EXIF as a phone writes it: the maker, the orientation and
// the place, in the byte order given.
func tiffWithPlace(order binary.ByteOrder) []byte {
	b := make([]byte, 158)
	if order == binary.BigEndian {
		copy(b, "MM")
	} else {
		copy(b, "II")
	}
	order.PutUint16(b[2:], 42)
	order.PutUint32(b[4:], 8)
	entry := func(at int, tag, typ uint16, count, value uint32) {
		order.PutUint16(b[at:], tag)
		order.PutUint16(b[at+2:], typ)
		order.PutUint32(b[at+4:], count)
		order.PutUint32(b[at+8:], value)
	}
	order.PutUint16(b[8:], 3)
	entry(10, 0x010F, 2, 6, 50) // Make, out of line
	entry(22, 0x0112, 3, 1, 0)  // Orientation
	order.PutUint16(b[30:], 6)  // a short sits at the start of its value
	entry(34, 0x8825, 4, 1, 56) // GPSInfo
	copy(b[50:], "Apple\x00")   // the maker
	order.PutUint16(b[56:], 4)  // the GPS directory
	entry(58, 0x0001, 2, 2, 0)  // GPSLatitudeRef
	copy(b[66:], "N")
	entry(70, 0x0002, 5, 3, 110) // GPSLatitude
	entry(82, 0x0003, 2, 2, 0)   // GPSLongitudeRef
	copy(b[90:], "E")
	entry(94, 0x0004, 5, 3, 134) // GPSLongitude
	copy(b[110:], latitude)
	copy(b[134:], latitude)
	return b
}

// placedXMP names a place in the ways XMP can, as an attribute, as an
// element and as a drone writes it, with the rest of the GPS properties an
// editor copies from EXIF, among what else a photo's XMP says.
const placedXMP = `<x:xmpmeta xmlns:x="adobe:ns:meta/"><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
	`<rdf:Description rdf:about="" xmlns:exif="http://ns.adobe.com/exif/1.0/" xmlns:xmp="http://ns.adobe.com/xap/1.0/"` +
	` xmlns:drone-dji="http://www.dji.com/drone-dji/1.0/" xmlns:dc="http://purl.org/dc/elements/1.1/"` +
	` xmlns:gps="http://example.com/gps/" exif:GPSLatitude="31,13.752N" xmp:Rating="5" drone-dji:GpsLongtitude='+121.4737'` +
	` exif:GPSAltitude="25/2" drone-dji:RelativeAltitude="+80.20">` +
	`<exif:GPSLongitude>121,28.422E</exif:GPSLongitude><exif:GPSTimeStamp>2026-09-29T08:30:00Z</exif:GPSTimeStamp>` +
	`<dc:description><rdf:Alt><rdf:li xml:lang="x-default">The latitude of the river mouth</rdf:li></rdf:Alt></dc:description>` +
	`</rdf:Description></rdf:RDF></x:xmpmeta>`

// coordinates are the parts of placedXMP that say where.
var coordinates = []string{"31,13.752N", "121,28.422E", "+121.4737", "25/2", "08:30:00Z", "GPSLatitude", "GPSLongitude", "GpsLongtitude"}

// wellFormed reports whether XML reads through to its end.
func wellFormed(t *testing.T, data []byte) {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		_, err := d.Token()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Fatalf("the XMP is no longer well formed: %v\n%s", err, data)
		}
	}
}

// withSegments puts JPEG segments right after a picture's start.
func withSegments(jpeg []byte, segments ...[]byte) []byte {
	out := append([]byte(nil), jpeg[:2]...)
	for _, s := range segments {
		out = append(out, s...)
	}
	return append(out, jpeg[2:]...)
}

func app1(body []byte) []byte {
	return append([]byte{0xFF, 0xE1, byte((len(body) + 2) >> 8), byte(len(body) + 2)}, body...)
}

func exifSegment(tiff []byte) []byte { return app1(append([]byte("Exif\x00\x00"), tiff...)) }

func xmpSegment(xml string) []byte {
	return app1(append([]byte("http://ns.adobe.com/xap/1.0/\x00"), xml...))
}

// placeLeft names what of a place a picture still holds, if anything.
func placeLeft(data []byte) string {
	if bytes.Contains(data, latitude) {
		return "the latitude"
	}
	for _, c := range coordinates {
		if bytes.Contains(data, []byte(c)) {
			return c
		}
	}
	return ""
}

// once checks that a picture with nothing left to take out comes back as it
// is.
func once(t *testing.T, out []byte) {
	t.Helper()
	if again, removed := WithoutLocation(out); removed || !bytes.Equal(again, out) {
		t.Error("a picture with no place left was changed again")
	}
}

// A photo's place is taken out and everything else it says stays: how to
// turn it and what took it, in either byte order, with nothing in the file
// moved.
func TestAPhotosPlaceIsTakenOut(t *testing.T) {
	for _, order := range []binary.ByteOrder{binary.BigEndian, binary.LittleEndian} {
		src := withSegments(picture(t, 40, 20), exifSegment(tiffWithPlace(order)))
		out, removed := WithoutLocation(src)
		if !removed {
			t.Fatalf("%v: nothing was taken out", order)
		}
		if left := placeLeft(out); left != "" {
			t.Errorf("%v: %s is still in the file", order, left)
		}
		if orientation(out) != 6 || !bytes.Contains(out, []byte("Apple\x00")) {
			t.Errorf("%v: the orientation or the maker went with the place", order)
		}
		if len(out) != len(src) {
			t.Errorf("%v: %d bytes became %d", order, len(src), len(out))
		}
		if _, _, err := image.Decode(bytes.NewReader(out)); err != nil {
			t.Errorf("%v: the photo no longer decodes: %v", order, err)
		}
		once(t, out)
	}
}

// XMP loses the properties that say where and nothing else: not what else it
// says, not a word in its text, and not its length, and it stays XML.
func TestXMPLosesOnlyItsCoordinates(t *testing.T) {
	xmp := []byte(placedXMP)
	if !blankPlaces(xmp) {
		t.Fatal("nothing was taken out")
	}
	if left := placeLeft(xmp); left != "" {
		t.Errorf("%s is still there:\n%s", left, xmp)
	}
	for _, kept := range []string{`xmp:Rating="5"`, "The latitude of the river mouth", `rdf:about=""`,
		`xmlns:gps="http://example.com/gps/"`, `drone-dji:RelativeAltitude="+80.20"`} {
		if !bytes.Contains(xmp, []byte(kept)) {
			t.Errorf("%s went with the place", kept)
		}
	}
	if len(xmp) != len(placedXMP) {
		t.Errorf("%d bytes became %d", len(placedXMP), len(xmp))
	}
	wellFormed(t, xmp)

	for _, plain := range []string{
		`<x:xmpmeta><rdf:Description xmp:Rating="5"/></x:xmpmeta>`,
		`<dc:title>Latitude: a novel</dc:title>`,
	} {
		if xmp := []byte(plain); blankPlaces(xmp) || string(xmp) != plain {
			t.Errorf("XMP that names no place was changed: %s", xmp)
		}
	}

	// A property that is not closed leaves the packet unreadable, so all of
	// it goes.
	broken := []byte(`<rdf:Description><exif:GPSLatitude>31,13.752N</rdf:Description>`)
	if !blankPlaces(broken) || strings.TrimSpace(string(broken)) != "" {
		t.Errorf("an unreadable packet was left as %q", broken)
	}
}

// A JPEG's XMP is cleaned where it lies, and one that names no place is left
// as it is, as is a file that is no picture.
func TestAJPEGsXMPIsCleanedWhereItLies(t *testing.T) {
	src := withSegments(picture(t, 40, 20), xmpSegment(placedXMP))
	out, removed := WithoutLocation(src)
	if !removed || placeLeft(out) != "" || len(out) != len(src) || !bytes.Contains(out, []byte(`xmp:Rating="5"`)) {
		t.Errorf("removed %v, left %q, %d bytes of %d", removed, placeLeft(out), len(out), len(src))
	}
	once(t, out)

	unplaced := tiffWithPlace(binary.BigEndian)
	binary.BigEndian.PutUint16(unplaced[8:], 2) // the GPS pointer is the third entry
	src = withSegments(picture(t, 40, 20), exifSegment(unplaced), xmpSegment(`<x:xmpmeta xmp:Rating="5"/>`))
	if out, removed := WithoutLocation(src); removed || !bytes.Equal(out, src) {
		t.Error("a photo that names no place was changed")
	}
	if out, removed := WithoutLocation([]byte("%PDF-1.7")); removed || string(out) != "%PDF-1.7" {
		t.Error("a file that is not a picture was changed")
	}
}

// EXIF that cannot be read may hold a place nobody can find, so it is zeroed
// whole where it lies, and the photo still decodes.
func TestUnreadableEXIFGoesWhole(t *testing.T) {
	broken := tiffWithPlace(binary.BigEndian)
	binary.BigEndian.PutUint32(broken[4:], 1<<31) // a first directory past the end
	src := withSegments(picture(t, 40, 20), exifSegment(broken))
	out, removed := WithoutLocation(src)
	if !removed || bytes.Contains(out, []byte("Exif\x00\x00")) || placeLeft(out) != "" || len(out) != len(src) {
		t.Errorf("removed %v, EXIF left %v, %d bytes of %d", removed, bytes.Contains(out, []byte("Exif\x00\x00")), len(out), len(src))
	}
	if _, _, err := image.Decode(bytes.NewReader(out)); err != nil {
		t.Errorf("the photo no longer decodes: %v", err)
	}
	once(t, out)
}

// Some cameras keep a larger preview after the photo, with EXIF of its own,
// found by an offset from the photo's start: it loses its place too, and
// nothing moves.
func TestAPictureAfterThePhotoLosesItsPlace(t *testing.T) {
	preview := withSegments(picture(t, 20, 10), exifSegment(tiffWithPlace(binary.LittleEndian)))
	src := append(withSegments(picture(t, 40, 20), exifSegment(tiffWithPlace(binary.BigEndian))), preview...)
	out, removed := WithoutLocation(src)
	if !removed || placeLeft(out) != "" || len(out) != len(src) {
		t.Fatalf("removed %v, left %q, %d bytes of %d", removed, placeLeft(out), len(out), len(src))
	}
	at := len(src) - len(preview)
	if orientation(out[at:]) != 6 {
		t.Error("the preview lost its orientation")
	}
	if _, _, err := image.Decode(bytes.NewReader(out[at:])); err != nil {
		t.Errorf("the preview no longer decodes: %v", err)
	}
}

func pngChunk(kind string, data []byte) []byte {
	out := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
	out = append(out, kind...)
	out = append(out, data...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(append([]byte(kind), data...)))
}

func deflated(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// imageMagickProfile is a profile as ImageMagick keeps one in a PNG's text.
func imageMagickProfile(name string, data []byte) []byte {
	digits := hex.EncodeToString(data)
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s\n%8d\n", name, len(data))
	for len(digits) > 72 {
		b.WriteString(digits[:72] + "\n")
		digits = digits[72:]
	}
	b.WriteString(digits + "\n")
	return []byte(b.String())
}

// A PNG's place goes from each place a PNG keeps one: its EXIF, cleaned with
// its checksum made again, which a decoder checks; its XMP, as it is or
// compressed; and the EXIF and XMP a converter kept as text, whole or a
// property at a time. Text that names no place stays.
func TestAPNGsPlaceIsTakenOut(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	plain := buf.Bytes()
	ihdrEnd := 8 + 8 + 13 + 4
	var chunks []byte
	for _, c := range [][]byte{
		pngChunk("eXIf", tiffWithPlace(binary.BigEndian)),
		pngChunk("iTXt", append([]byte("XML:com.adobe.xmp\x00\x00\x00\x00\x00"), placedXMP...)),
		pngChunk("iTXt", append([]byte("XML:com.adobe.xmp\x00\x01\x00\x00\x00"), deflated(t, []byte(placedXMP))...)),
		pngChunk("zTXt", append([]byte("Raw profile type exif\x00\x00"),
			deflated(t, imageMagickProfile("exif", append([]byte("Exif\x00\x00"), tiffWithPlace(binary.LittleEndian)...)))...)),
		pngChunk("tEXt", append([]byte("Raw profile type xmp\x00"), imageMagickProfile("xmp", []byte(placedXMP))...)),
		pngChunk("tEXt", []byte("exif:GPSLatitude\x0031/1,13/1,4512/100")),
		pngChunk("tEXt", []byte("exif:Make\x00Apple")),
		pngChunk("tEXt", []byte("Comment\x00taken at the latitude of home")),
	} {
		chunks = append(chunks, c...)
	}
	src := append(append(append([]byte(nil), plain[:ihdrEnd]...), chunks...), plain[ihdrEnd:]...)

	out, removed := WithoutLocation(src)
	if !removed {
		t.Fatal("nothing was taken out")
	}
	if left := placeLeft(out); left != "" {
		t.Errorf("%s is still there", left)
	}
	for _, kept := range []string{"eXIf", "Apple\x00", `xmp:Rating="5"`, "taken at the latitude of home"} {
		if !bytes.Contains(out, []byte(kept)) {
			t.Errorf("%q went with the place", kept)
		}
	}
	if got, want := pngChunks(out), "IHDR eXIf iTXt:XML:com.adobe.xmp tEXt:exif:Make tEXt:Comment IDAT IEND"; got != want {
		t.Errorf("chunks are %s, want %s", got, want)
	}
	if _, err := png.Decode(bytes.NewReader(out)); err != nil {
		t.Errorf("the PNG no longer decodes: %v", err)
	}
	once(t, out)
}

// pngChunks lists a PNG's chunks, with the keyword of each text.
func pngChunks(data []byte) string {
	var kinds []string
	for i := 8; i+12 <= len(data); {
		n := int(binary.BigEndian.Uint32(data[i:]))
		kind := string(data[i+4 : i+8])
		if kind == "iTXt" || kind == "tEXt" || kind == "zTXt" {
			keyword, _, _ := bytes.Cut(data[i+8:i+8+n], []byte{0})
			kind += ":" + string(keyword)
		}
		kinds = append(kinds, kind)
		i += 12 + n
	}
	return strings.Join(kinds, " ")
}

func webpChunk(kind string, data []byte) []byte {
	out := append([]byte(kind), binary.LittleEndian.AppendUint32(nil, uint32(len(data)))...)
	out = append(out, data...)
	if len(data)%2 == 1 {
		out = append(out, 0)
	}
	return out
}

// extendedWebP is a WebP in the extended format, which is where EXIF and
// XMP go, around an opaque picture.
func extendedWebP(t *testing.T, flags byte, chunks ...[]byte) []byte {
	t.Helper()
	opaque := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := range opaque.Pix {
		opaque.Pix[i] = 0xFF
	}
	var simple bytes.Buffer
	if err := webp.Encode(&simple, opaque, &webp.EncoderOptions{Quality: 75}); err != nil {
		t.Fatal(err)
	}
	vp8 := simple.Bytes()[12:] // the image chunk of a simple WebP
	if string(vp8[:4]) != "VP8 " {
		t.Fatalf("a simple WebP starts with %q", vp8[:4])
	}
	body := append([]byte("WEBP"), webpChunk("VP8X", []byte{flags, 0, 0, 0, 3, 0, 0, 3, 0, 0})...)
	body = append(body, vp8...)
	for _, c := range chunks {
		body = append(body, c...)
	}
	return append(append([]byte("RIFF"), binary.LittleEndian.AppendUint32(nil, uint32(len(body)))...), body...)
}

// A WebP's EXIF and XMP are cleaned where they lie, and EXIF that cannot be
// read goes, with its flag and the file's size saying so.
func TestAWebPsPlaceIsTakenOut(t *testing.T) {
	src := extendedWebP(t, 0x08|0x04,
		webpChunk("EXIF", tiffWithPlace(binary.LittleEndian)),
		webpChunk("XMP ", []byte(placedXMP)))
	out, removed := WithoutLocation(src)
	if !removed || placeLeft(out) != "" || len(out) != len(src) || out[20] != 0x08|0x04 {
		t.Fatalf("removed %v, left %q, %d bytes of %d, flags %#x", removed, placeLeft(out), len(out), len(src), out[20])
	}
	if _, err := webp.Decode(bytes.NewReader(out)); err != nil {
		t.Errorf("the WebP no longer decodes: %v", err)
	}
	once(t, out)

	broken := tiffWithPlace(binary.LittleEndian)
	binary.LittleEndian.PutUint16(broken[2:], 43)
	src = extendedWebP(t, 0x08, webpChunk("EXIF", broken))
	out, removed = WithoutLocation(src)
	if !removed || bytes.Contains(out, []byte("EXIF")) || out[20] != 0 {
		t.Errorf("removed %v, EXIF kept %v, flags %#x", removed, bytes.Contains(out, []byte("EXIF")), out[20])
	}
	if size := binary.LittleEndian.Uint32(out[4:]); int(size) != len(out)-8 {
		t.Errorf("RIFF size %d for %d bytes", size, len(out))
	}
	if _, err := webp.Decode(bytes.NewReader(out)); err != nil {
		t.Errorf("the WebP no longer decodes: %v", err)
	}
}

func isoBox(kind string, payload ...[]byte) []byte {
	body := bytes.Join(payload, nil)
	return append(binary.BigEndian.AppendUint32(nil, uint32(8+len(body))), append([]byte(kind), body...)...)
}

func u16(v int) []byte { return binary.BigEndian.AppendUint16(nil, uint16(v)) }
func u32(v int) []byte { return binary.BigEndian.AppendUint32(nil, uint32(v)) }

// avif is an AVIF as libheif writes one, less the picture: its EXIF item in
// the file and its XMP item in the meta box's own data, the way an iloc of
// version 1 can say.
func avif(exif, xmp []byte) []byte {
	ftyp := isoBox("ftyp", []byte("avif"), u32(0), []byte("mif1miafavif"))
	infe := func(id int, typ string, extra string) []byte {
		return isoBox("infe", []byte{2, 0, 0, 0}, u16(id), u16(0), []byte(typ), []byte("\x00"+extra))
	}
	iinf := isoBox("iinf", []byte{0, 0, 0, 0}, u16(3),
		infe(1, "av01", ""), infe(2, "Exif", ""), infe(3, "mime", "application/rdf+xml\x00"))
	idat := isoBox("idat", xmp)
	iloc := func(exifAt, xmpAt int) []byte {
		return isoBox("iloc", []byte{1, 0, 0, 0, 0x44, 0x00}, u16(2),
			u16(2), u16(0), u16(0), u16(1), u32(exifAt), u32(len(exif)), // in the file
			u16(3), u16(1), u16(0), u16(1), u32(0), u32(len(xmp)), // in idat
		)
	}
	hdlr := isoBox("hdlr", make([]byte, 8), []byte("pict"), make([]byte, 13))
	meta := func(exifAt int) []byte {
		return isoBox("meta", []byte{0, 0, 0, 0}, hdlr, iinf, iloc(exifAt, 0), idat)
	}
	// Where the EXIF lands depends on the size of what comes before it,
	// which does not depend on where it lands.
	exifAt := len(ftyp) + len(meta(0)) + 8
	mdat := isoBox("mdat", exif, []byte("the picture itself"))
	return bytes.Join([][]byte{ftyp, meta(exifAt), mdat}, nil)
}

// exifItem is an EXIF item's bytes: how far to skip, the header a JPEG
// would have, and the TIFF data.
func exifItem(tiff []byte) []byte {
	return append(append(u32(6), "Exif\x00\x00"...), tiff...)
}

// An AVIF's EXIF and XMP items are found where its item locations say, in
// the file or in the meta box, and cleaned where they lie.
func TestAnAVIFsPlaceIsTakenOut(t *testing.T) {
	src := avif(exifItem(tiffWithPlace(binary.BigEndian)), []byte(placedXMP))
	if !isHEIF(src) {
		t.Fatal("an AVIF was not known as one")
	}
	out, removed := WithoutLocation(src)
	if !removed || placeLeft(out) != "" || len(out) != len(src) {
		t.Fatalf("removed %v, left %q, %d bytes of %d", removed, placeLeft(out), len(out), len(src))
	}
	// The item's own header, not the item type the list names it by.
	at := bytes.Index(out, []byte("Exif\x00\x00MM")) + 6
	if at < 6 || exifOrientation(out[at:]) != 6 || !bytes.Contains(out, []byte("Apple\x00")) || !bytes.Contains(out, []byte(`xmp:Rating="5"`)) {
		t.Error("what else the AVIF said went with the place")
	}
	once(t, out)

	broken := tiffWithPlace(binary.BigEndian)
	copy(broken, "XX")
	out, removed = WithoutLocation(avif(exifItem(broken), []byte(`<x:xmpmeta/>`)))
	if !removed || bytes.Contains(out, []byte("Apple\x00")) || placeLeft(out) != "" {
		t.Errorf("unreadable EXIF: removed %v, maker kept %v, left %q", removed, bytes.Contains(out, []byte("Apple\x00")), placeLeft(out))
	}
	once(t, out)
}

// No file, however it is made, makes taking its place out fail, and a JPEG
// or an AVIF never changes its length.
func FuzzWithoutLocation(f *testing.F) {
	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	for _, seed := range [][]byte{
		withSegments(picture(f, 8, 8), exifSegment(tiffWithPlace(binary.BigEndian)), xmpSegment(placedXMP)),
		append(buf.Bytes()[:33:33], pngChunk("eXIf", tiffWithPlace(binary.LittleEndian))...),
		append([]byte("RIFF\x00\x00\x00\x00WEBP"), webpChunk("EXIF", tiffWithPlace(binary.LittleEndian))...),
		avif(exifItem(tiffWithPlace(binary.BigEndian)), []byte(placedXMP)),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		src := bytes.Clone(data)
		out, _ := WithoutLocation(data)
		if !bytes.Equal(data, src) {
			t.Fatal("the bytes given were changed")
		}
		if (bytes.HasPrefix(data, []byte{0xFF, 0xD8}) || isHEIF(data)) && len(out) != len(data) {
			t.Fatalf("%d bytes became %d", len(data), len(out))
		}
		WithoutLocation(out)
	})
}
