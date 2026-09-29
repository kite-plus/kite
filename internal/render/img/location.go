package img

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"hash/crc32"
	"io"
	"math"
	"strconv"
	"strings"
)

// WithoutLocation takes out where a picture was taken, which a phone writes
// into every photo, and reports whether there was any: the GPS directory of
// its EXIF and the GPS properties of its XMP, wherever the format keeps
// them. Everything else stays, the orientation above all, which says how to
// turn the photo, and the pixels are never touched. JPEG, PNG, WebP and AVIF
// are read; anything else comes back as it is. EXIF that cannot be read goes
// whole, since what it holds cannot be known.
func WithoutLocation(data []byte) ([]byte, bool) {
	switch {
	case len(data) > 2 && data[0] == 0xFF && data[1] == 0xD8:
		return jpegWithoutLocation(data)
	case bytes.HasPrefix(data, pngSignature):
		return pngWithoutLocation(data)
	case len(data) > 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return webpWithoutLocation(data)
	case isHEIF(data):
		return heifWithoutLocation(data)
	}
	return data, false
}

var (
	pngSignature = []byte("\x89PNG\r\n\x1a\n")
	exifHeader   = []byte("Exif\x00\x00")
	xmpHeader    = []byte("http://ns.adobe.com/xap/1.0/\x00")
	// An extension carries on XMP too long for one block, after a GUID, the
	// full length and an offset.
	xmpExtension = []byte("http://ns.adobe.com/xmp/extension/\x00")
)

// jpegWithoutLocation cleans every EXIF and XMP block of a JPEG where it
// lies, the photo's own and those of the pictures some cameras keep after
// it, so that nothing moves: those pictures are found by their offsets.
func jpegWithoutLocation(data []byte) ([]byte, bool) {
	var out []byte // a copy, made at the first change
	for i := 0; ; {
		j := bytes.Index(data[i:], []byte{0xFF, 0xE1})
		if j < 0 {
			break
		}
		at := i + j
		i = at + 2
		if at+4 > len(data) {
			break
		}
		end := at + 2 + int(binary.BigEndian.Uint16(data[at+2:]))
		if end < at+4 || end > len(data) {
			continue
		}
		from := data
		if out != nil {
			from = out
		}
		block := from[at+4 : end]
		if !bytes.HasPrefix(block, exifHeader) && !bytes.HasPrefix(block, xmpHeader) && !bytes.HasPrefix(block, xmpExtension) {
			continue
		}
		block = bytes.Clone(block)
		if !cleanAPP1(block) {
			continue
		}
		if out == nil {
			out = bytes.Clone(data)
		}
		copy(out[at+4:end], block)
	}
	if out == nil {
		return data, false
	}
	return out, true
}

// cleanAPP1 takes the place out of an APP1 block where it lies, EXIF or XMP,
// and reports whether it changed it.
func cleanAPP1(block []byte) bool {
	switch {
	case bytes.HasPrefix(block, exifHeader):
		removed, ok := withoutGPS(block[len(exifHeader):])
		if !ok {
			clear(block)
			return true
		}
		return removed
	case bytes.HasPrefix(block, xmpHeader):
		return blankPlaces(block[len(xmpHeader):])
	case bytes.HasPrefix(block, xmpExtension) && len(block) >= len(xmpExtension)+40:
		return blankPlaces(block[len(xmpExtension)+40:])
	}
	return false
}

func pngWithoutLocation(data []byte) ([]byte, bool) {
	out := append([]byte(nil), data[:8]...)
	removed := false
	i := 8
	for i+12 <= len(data) {
		size := uint64(binary.BigEndian.Uint32(data[i:]))
		if !fits(uint64(i)+12, size, len(data)) {
			break
		}
		chunk := data[i : i+12+int(size)]
		i += len(chunk)
		kind := string(chunk[4:8])
		cleaned, keep := pngChunkWithoutLocation(kind, chunk)
		switch {
		case !keep:
			removed = true
			continue
		case cleaned != nil:
			removed = true
			chunk = cleaned
		}
		out = append(out, chunk...)
		if kind == "IEND" {
			break
		}
	}
	if !removed {
		return data, false
	}
	return append(out, data[i:]...), true
}

// pngChunkWithoutLocation returns a PNG chunk cleaned, nil for one that named
// no place, or keep false for one that goes whole.
func pngChunkWithoutLocation(kind string, chunk []byte) (cleaned []byte, keep bool) {
	body := chunk[8 : len(chunk)-4]
	switch kind {
	case "eXIf":
		cleaned = bytes.Clone(chunk)
		// Some writers keep JPEG's Exif header before the TIFF data.
		removed, ok := withoutGPS(bytes.TrimPrefix(cleaned[8:len(cleaned)-4], exifHeader))
		if !ok {
			return nil, false
		}
		if !removed {
			return nil, true
		}
	case "iTXt", "tEXt", "zTXt":
		k := bytes.IndexByte(body, 0)
		if k < 0 {
			return nil, true
		}
		keyword := string(body[:k])
		switch {
		case placeKeyword(keyword):
			// ImageMagick writes each property of a photo's EXIF as a text
			// of its own, exif:GPSLatitude among them.
			return nil, false
		case keyword == "XML:com.adobe.xmp":
			text, at, ok := pngText(kind, body, k)
			switch {
			case !ok:
				return nil, false
			case at < 0:
				// Compressed, it is taken out rather than written again.
				return nil, !blankPlaces(text)
			}
			cleaned = bytes.Clone(chunk)
			if !blankPlaces(cleaned[8+at : len(cleaned)-4]) {
				return nil, true
			}
		case strings.HasPrefix(strings.ToLower(keyword), "raw profile type "):
			// EXIF and XMP as ImageMagick keeps them, in hex, which no
			// browser reads; one that names a place is taken out.
			profile := strings.ToLower(keyword[len("raw profile type "):])
			if profile != "exif" && profile != "app1" && profile != "xmp" {
				return nil, true
			}
			text, _, ok := pngText(kind, body, k)
			if !ok {
				return nil, false
			}
			raw, ok := rawProfile(text)
			return nil, ok && !profileNamesPlace(profile, raw)
		default:
			return nil, true
		}
	default:
		return nil, true
	}
	binary.BigEndian.PutUint32(cleaned[len(cleaned)-4:], crc32.ChecksumIEEE(cleaned[4:len(cleaned)-4]))
	return cleaned, true
}

// placeKeyword reports whether a PNG text's keyword says it holds where a
// picture was taken.
func placeKeyword(keyword string) bool {
	k := []byte(strings.ToLower(keyword))
	if bytes.Contains(k, []byte("gps")) {
		return true
	}
	for _, word := range placeWords {
		if bytes.Contains(k, word) {
			return true
		}
	}
	return false
}

// maxText bounds what a compressed PNG text is inflated to.
const maxText = 16 << 20

// pngText returns the text of a PNG text chunk whose keyword ends at k, and
// where in the chunk's body it lies, or -1 when it was compressed.
func pngText(kind string, body []byte, k int) (text []byte, at int, ok bool) {
	p := k + 1
	switch kind {
	case "tEXt":
		return body[p:], p, true
	case "zTXt":
		if p >= len(body) {
			return nil, -1, false
		}
		text, ok = inflate(body[p+1:])
		return text, -1, ok
	}
	// iTXt: whether it is compressed and how, a language and the keyword
	// translated, then the text.
	if p+2 > len(body) {
		return nil, -1, false
	}
	compressed := body[p] == 1
	p += 2
	for range 2 {
		n := bytes.IndexByte(body[p:], 0)
		if n < 0 {
			return nil, -1, false
		}
		p += n + 1
	}
	if !compressed {
		return body[p:], p, true
	}
	text, ok = inflate(body[p:])
	return text, -1, ok
}

func inflate(data []byte) ([]byte, bool) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	text, err := io.ReadAll(io.LimitReader(r, maxText+1))
	return text, err == nil && len(text) <= maxText
}

// rawProfile decodes a profile as ImageMagick keeps one in a PNG's text: its
// name, its length and its bytes in hex.
func rawProfile(text []byte) ([]byte, bool) {
	fields := bytes.Fields(text)
	if len(fields) < 2 {
		return nil, false
	}
	n, err := strconv.Atoi(string(fields[1]))
	digits := bytes.Join(fields[2:], nil)
	if err != nil || len(digits)%2 != 0 || len(digits)/2 != n {
		return nil, false
	}
	raw := make([]byte, n)
	if _, err := hex.Decode(raw, digits); err != nil {
		return nil, false
	}
	return raw, true
}

// profileNamesPlace reports whether a raw profile names a place, or cannot
// be read.
func profileNamesPlace(profile string, raw []byte) bool {
	switch {
	case profile == "xmp":
		return blankPlaces(raw)
	case bytes.HasPrefix(raw, exifHeader) || bytes.HasPrefix(raw, xmpHeader) || bytes.HasPrefix(raw, xmpExtension):
		return cleanAPP1(raw)
	case profile == "exif":
		removed, ok := withoutGPS(raw)
		return removed || !ok
	}
	return false
}

func webpWithoutLocation(data []byte) ([]byte, bool) {
	out := append([]byte(nil), data[:12]...)
	var dropped byte // the VP8X flags of the chunks taken out
	removed := false
	i := 12
	for i+8 <= len(data) {
		declared := uint64(binary.LittleEndian.Uint32(data[i+4:]))
		if !fits(uint64(i)+8, declared+declared&1, len(data)) {
			break
		}
		size := int(declared)
		chunk := data[i : i+8+size+size&1]
		i += len(chunk)
		switch string(chunk[:4]) {
		case "EXIF":
			cleaned := bytes.Clone(chunk)
			// Some writers keep JPEG's Exif header before the TIFF data.
			gone, ok := withoutGPS(bytes.TrimPrefix(cleaned[8:8+size], exifHeader))
			if !ok {
				removed, dropped = true, dropped|0x08
				continue
			}
			if gone {
				removed = true
				chunk = cleaned
			}
		case "XMP ":
			cleaned := bytes.Clone(chunk)
			if blankPlaces(cleaned[8 : 8+size]) {
				removed = true
				chunk = cleaned
			}
		}
		out = append(out, chunk...)
	}
	if !removed {
		return data, false
	}
	out = append(out, data[i:]...)
	if dropped != 0 && len(out) > 20 && string(out[12:16]) == "VP8X" {
		out[20] &^= dropped // the flags that say those chunks follow
	}
	binary.LittleEndian.PutUint32(out[4:], uint32(len(out)-8))
	return out, true
}

// isHEIF reports whether data is a HEIF picture, which AVIF is: an ISO base
// media file whose brands say it holds pictures.
func isHEIF(data []byte) bool {
	if len(data) < 16 || string(data[4:8]) != "ftyp" {
		return false
	}
	size := uint64(binary.BigEndian.Uint32(data))
	if size < 16 || !fits(0, size, len(data)) {
		return false
	}
	pictures := func(brand []byte) bool {
		switch string(brand) {
		case "mif1", "msf1", "avif", "avis", "heic", "heix":
			return true
		}
		return false
	}
	if pictures(data[8:12]) {
		return true
	}
	for p := uint64(16); p+4 <= size; p += 4 {
		if pictures(data[p : p+4]) {
			return true
		}
	}
	return false
}

// heifWithoutLocation cleans a HEIF picture's EXIF and XMP items where they
// lie, found through the meta box's item list and item locations, so that no
// offset in the file changes.
func heifWithoutLocation(data []byte) ([]byte, bool) {
	var meta box
	for _, b := range boxes(data, 0, uint64(len(data))) {
		if b.kind == "meta" {
			meta = b
			break
		}
	}
	if meta.kind == "" || meta.end-meta.start < 4 {
		return data, false
	}
	var kinds map[uint32]string
	var items []item
	var idat box
	ok := true
	// meta is a full box: a version and flags come first.
	for _, b := range boxes(data, meta.start+4, meta.end) {
		switch b.kind {
		case "iinf":
			kinds, ok = readItemKinds(data[b.start:b.end])
		case "iloc":
			items, ok = readItemLocations(data[b.start:b.end])
		case "idat":
			idat = b
		}
		if !ok {
			return data, false
		}
	}

	var out []byte // a copy, made at the first change
	for _, it := range items {
		kind := kinds[it.id]
		if kind == "" {
			continue
		}
		spans, ok := it.spans(idat, len(data))
		if !ok {
			continue
		}
		from := data
		if out != nil {
			from = out
		}
		var buf []byte
		for _, s := range spans {
			buf = append(buf, from[s.off:s.off+s.n]...)
		}
		if !cleanItem(kind, buf) {
			continue
		}
		if out == nil {
			out = bytes.Clone(data)
		}
		rest := buf
		for _, s := range spans {
			rest = rest[copy(out[s.off:s.off+s.n], rest):]
		}
	}
	if out == nil {
		return data, false
	}
	return out, true
}

// cleanItem takes the place out of a HEIF item where it lies and reports
// whether it changed it. An EXIF item starts with how far past that its TIFF
// data begins.
func cleanItem(kind string, it []byte) bool {
	if kind == "xmp" {
		return blankPlaces(it)
	}
	if len(it) >= 4 {
		if skip := uint64(binary.BigEndian.Uint32(it)); fits(4, skip, len(it)) {
			// Some writers keep JPEG's Exif header before the TIFF data.
			removed, ok := withoutGPS(bytes.TrimPrefix(it[4+skip:], exifHeader))
			if ok {
				return removed
			}
		}
	}
	if zeroed(it) {
		return false
	}
	clear(it)
	return true
}

// zeroed reports whether b holds nothing but zeros.
func zeroed(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}

// box is a box of an ISO base media file: its type and where what it holds
// lies.
type box struct {
	kind       string
	start, end uint64
}

// boxes reads the boxes that fill data[from:to], up to any it cannot read,
// such as bytes some tool left after the last.
func boxes(data []byte, from, to uint64) []box {
	var out []box
	for p := from; p < to; {
		if !fits(p, 8, int(to)) {
			break
		}
		size, header := uint64(binary.BigEndian.Uint32(data[p:])), uint64(8)
		switch size {
		case 0: // to the end
			size = to - p
		case 1: // a 64-bit size follows
			if !fits(p, 16, int(to)) {
				return out
			}
			size, header = binary.BigEndian.Uint64(data[p+8:]), 16
		}
		if size < header || !fits(p, size, int(to)) {
			break
		}
		out = append(out, box{kind: string(data[p+4 : p+8]), start: p + header, end: p + size})
		p += size
	}
	return out
}

// readItemKinds reads an iinf box for the items that can say where a picture
// was taken: "exif" and "xmp" by item id.
func readItemKinds(b []byte) (map[uint32]string, bool) {
	r := &reader{data: b}
	version := r.u8()
	r.skip(3)
	if version == 0 {
		r.u16()
	} else {
		r.u32()
	}
	if r.bad {
		return nil, false
	}
	kinds := map[uint32]string{}
	for _, e := range boxes(b, uint64(r.at), uint64(len(b))) {
		if e.kind != "infe" {
			continue
		}
		r := &reader{data: b[e.start:e.end]}
		version := r.u8()
		r.skip(3)
		if version < 2 { // no item types before version 2
			continue
		}
		var id uint32
		if version == 2 {
			id = uint32(r.u16())
		} else {
			id = r.u32()
		}
		r.skip(2) // protection
		typ := string(r.take(4))
		if r.bad {
			continue
		}
		switch typ {
		case "Exif":
			kinds[id] = "exif"
		case "mime":
			// A name, then the content type, each ending in a zero.
			_, rest, _ := bytes.Cut(r.data[r.at:], []byte{0})
			if content, _, _ := bytes.Cut(rest, []byte{0}); string(content) == "application/rdf+xml" {
				kinds[id] = "xmp"
			}
		}
	}
	return kinds, true
}

// item is where an item's bytes lie, by an iloc box.
type item struct {
	id      uint32
	method  uint16 // 0 in the file, 1 in the idat box
	extents []span
}

type span struct{ off, n uint64 }

// readItemLocations reads an iloc box. Items kept in other files are left
// out.
func readItemLocations(b []byte) ([]item, bool) {
	r := &reader{data: b}
	version := r.u8()
	r.skip(3)
	sizes, more := r.u8(), r.u8()
	offsetSize, lengthSize, baseSize, indexSize := sizes>>4, sizes&15, more>>4, more&15
	if version == 0 {
		indexSize = 0
	}
	var count uint32
	if version < 2 {
		count = uint32(r.u16())
	} else {
		count = r.u32()
	}
	var items []item
	for range count {
		var it item
		if version < 2 {
			it.id = uint32(r.u16())
		} else {
			it.id = r.u32()
		}
		if version == 1 || version == 2 {
			it.method = r.u16() & 15
		}
		ref := r.u16()
		base := r.sized(baseSize)
		extents := r.u16()
		for range extents {
			r.sized(indexSize)
			off, n := r.sized(offsetSize), r.sized(lengthSize)
			if off > math.MaxUint64-base {
				r.bad = true
			}
			it.extents = append(it.extents, span{base + off, n})
			if r.bad {
				return nil, false
			}
		}
		if r.bad {
			return nil, false
		}
		if ref == 0 {
			items = append(items, it)
		}
	}
	return items, !r.bad
}

// spans are where an item's bytes lie in a file of total bytes. An extent
// of no length runs to the end of what holds it.
func (it item) spans(idat box, total int) ([]span, bool) {
	var from, to uint64
	switch {
	case it.method == 0:
		from, to = 0, uint64(total)
	case it.method == 1 && idat.kind != "":
		from, to = idat.start, idat.end
	default:
		return nil, false
	}
	var out []span
	sum := uint64(0)
	for _, e := range it.extents {
		if e.off > to-from {
			return nil, false
		}
		off := from + e.off
		n := e.n
		if n == 0 {
			n = to - off
		}
		if !fits(off, n, int(to)) {
			return nil, false
		}
		// Extents that overlap cannot make an item larger than its file.
		if sum += n; sum > uint64(total) {
			return nil, false
		}
		out = append(out, span{off, n})
	}
	return out, true
}

// reader reads big-endian numbers from a box, and remembers running out.
type reader struct {
	data []byte
	at   int
	bad  bool
}

func (r *reader) take(n int) []byte {
	if r.bad || n > len(r.data)-r.at {
		r.bad = true
		return make([]byte, n)
	}
	r.at += n
	return r.data[r.at-n : r.at]
}

func (r *reader) skip(n int)  { r.take(n) }
func (r *reader) u8() byte    { return r.take(1)[0] }
func (r *reader) u16() uint16 { return binary.BigEndian.Uint16(r.take(2)) }
func (r *reader) u32() uint32 { return binary.BigEndian.Uint32(r.take(4)) }
func (r *reader) u64() uint64 { return binary.BigEndian.Uint64(r.take(8)) }

// sized reads a number of 0, 4 or 8 bytes, as iloc sizes its fields.
func (r *reader) sized(size byte) uint64 {
	switch size {
	case 0:
		return 0
	case 4:
		return uint64(r.u32())
	case 8:
		return r.u64()
	}
	r.bad = true
	return 0
}

// placeWords are what the names of a place's coordinates hold, in any case:
// exif:GPSLatitude, drone-dji:GpsLongitude and DJI's own misspelling,
// GpsLongtitude, among them.
var placeWords = [][]byte{[]byte("latitude"), []byte("longitude"), []byte("longtitude")}

// placeProperty reports whether an XMP property says where a picture was
// taken: any of EXIF's GPS properties, as the GPS directory goes whole from
// EXIF itself, or one named for a latitude or a longitude.
func placeProperty(name []byte) bool {
	prefix, local, ok := bytes.Cut(bytes.ToLower(name), []byte(":"))
	if !ok || string(prefix) == "xmlns" {
		return false
	}
	if bytes.HasPrefix(local, []byte("gps")) {
		return true
	}
	for _, word := range placeWords {
		if bytes.Contains(local, word) {
			return true
		}
	}
	return false
}

// searched are what lead to a place's property: its words, and the start of
// a GPS property's name.
var searched = append([][]byte{[]byte(":gps")}, placeWords...)

// blankPlaces overwrites with spaces each XMP property that says where a
// picture was taken, its element or its attribute whole, so the packet keeps
// its length and stays well formed. A property whose end cannot be found
// leaves the packet unreadable to it, and the whole packet is blanked. It
// reports whether anything was.
func blankPlaces(xmp []byte) bool {
	lower := bytes.Clone(xmp)
	for i, c := range lower {
		if 'A' <= c && c <= 'Z' {
			lower[i] = c + 'a' - 'A'
		}
	}
	// next is where each word is next found: -1 when it is not, and behind
	// the search when it has to be looked for again.
	next := make([]int, len(searched))
	for i := range next {
		next[i] = -2
	}
	changed := false
	for from := 0; ; {
		at, n := -1, 0
		for i, word := range searched {
			if next[i] != -1 && next[i] < from {
				if j := bytes.Index(lower[from:], word); j >= 0 {
					next[i] = from + j
				} else {
					next[i] = -1
				}
			}
			if next[i] >= 0 && (at < 0 || next[i] < at) {
				at, n = next[i], len(word)
			}
		}
		if at < 0 {
			return changed
		}
		start, end := at, at+n
		for start > 0 && nameByte(xmp[start-1]) {
			start--
		}
		for end < len(xmp) && nameByte(xmp[end]) {
			end++
		}
		from = end
		if !placeProperty(xmp[start:end]) {
			continue // a word in some text, or another property
		}
		lo, hi, ok := property(xmp, start, end)
		if !ok {
			fill(xmp)
			return true
		}
		if hi > lo {
			fill(xmp[lo:hi])
			changed = true
			from = hi
		}
	}
}

// property finds the whole of the property named at xmp[start:end]: the
// element it opens or the attribute it starts. The range is empty for a name
// that is neither, which is text, and ok is false for a property whose end
// cannot be found.
func property(xmp []byte, start, end int) (lo, hi int, ok bool) {
	switch {
	case start > 0 && xmp[start-1] == '<':
		gt := tagEnd(xmp, end)
		if gt < 0 {
			return 0, 0, false
		}
		if xmp[gt-1] == '/' {
			return start - 1, gt + 1, true
		}
		closing := append([]byte("</"), xmp[start:end]...)
		for c := gt; ; {
			i := bytes.Index(xmp[c:], closing)
			if i < 0 {
				return 0, 0, false
			}
			// Past a longer name that starts the same way.
			for c += i + len(closing); c < len(xmp) && space(xmp[c]); c++ {
			}
			if c < len(xmp) && xmp[c] == '>' {
				return start - 1, c + 1, true
			}
		}
	case start > 1 && xmp[start-1] == '/' && xmp[start-2] == '<':
		return 0, 0, false // the end of an element whose start was not seen
	case start == 0 || !space(xmp[start-1]):
		return 0, 0, true
	}
	p := end
	for p < len(xmp) && space(xmp[p]) {
		p++
	}
	if p >= len(xmp) || xmp[p] != '=' {
		return 0, 0, true
	}
	for p++; p < len(xmp) && space(xmp[p]); p++ {
	}
	if p >= len(xmp) || (xmp[p] != '"' && xmp[p] != '\'') {
		return 0, 0, false
	}
	q := bytes.IndexByte(xmp[p+1:], xmp[p])
	if q < 0 {
		return 0, 0, false
	}
	return start, p + q + 2, true
}

// tagEnd finds the > that ends a tag, past any quoted value.
func tagEnd(xmp []byte, from int) int {
	for p := from; p < len(xmp); p++ {
		switch xmp[p] {
		case '>':
			return p
		case '"', '\'':
			q := bytes.IndexByte(xmp[p+1:], xmp[p])
			if q < 0 {
				return -1
			}
			p += q + 1
		}
	}
	return -1
}

func nameByte(c byte) bool {
	return c >= 0x80 || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' ||
		c == ':' || c == '-' || c == '_' || c == '.'
}

func space(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

func fill(b []byte) {
	for i := range b {
		b[i] = ' '
	}
}

// withoutGPS takes the GPS directory out of TIFF data in place: its pointer
// in the first directory goes, and the directory and every value it points to
// are zeroed, so nothing of the place is left in the file and every other
// offset stays as it was. It reports whether there was one, and ok false when
// the data cannot be read. Offsets are worked in uint64, since they are
// 32-bit and a 32-bit build's int cannot hold them all.
func withoutGPS(tiff []byte) (removed, ok bool) {
	if len(tiff) < 8 {
		return false, false
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return false, false
	}
	if order.Uint16(tiff[2:]) != 42 {
		return false, false
	}
	ifd := uint64(order.Uint32(tiff[4:]))
	if ifd < 8 || !fits(ifd, 2, len(tiff)) {
		return false, false
	}
	n := uint64(order.Uint16(tiff[ifd:]))
	end := ifd + 2 + 12*n + 4
	if !fits(0, end, len(tiff)) {
		return false, false
	}
	k := n
	for e := range n {
		if order.Uint16(tiff[ifd+2+12*e:]) == 0x8825 {
			k = e
			break
		}
	}
	if k == n {
		return false, true
	}

	gps := uint64(order.Uint32(tiff[ifd+2+12*k+8:]))
	if gps >= 8 && fits(gps, 2, len(tiff)) {
		m := uint64(order.Uint16(tiff[gps:]))
		dir := gps + 2 + 12*m + 4
		if !fits(0, dir, len(tiff)) {
			return false, false
		}
		for e := range m {
			entry := tiff[gps+2+12*e:]
			width, known := typeWidth[order.Uint16(entry[2:])]
			if !known {
				return false, false
			}
			if size := width * uint64(order.Uint32(entry[4:])); size > 4 {
				at := uint64(order.Uint32(entry[8:]))
				if at < 8 || !fits(at, size, len(tiff)) {
					return false, false
				}
				clear(tiff[at : at+size])
			}
		}
		clear(tiff[gps:dir])
	}

	// The entries after the pointer move up over it, the offset of the next
	// directory with them, and what they leave at the end is zeroed.
	at := ifd + 2 + 12*k
	copy(tiff[at:end-12], tiff[at+12:end])
	clear(tiff[end-12 : end])
	order.PutUint16(tiff[ifd:], uint16(n-1))
	return true, true
}

// fits reports whether n bytes at off lie within total bytes, without the
// sum overflowing.
func fits(off, n uint64, total int) bool {
	t := uint64(total)
	return off <= t && n <= t-off
}

// typeWidth is how many bytes one value of each TIFF type takes.
var typeWidth = map[uint16]uint64{
	1: 1, 2: 1, 3: 2, 4: 4, 5: 8, 6: 1, 7: 1, 8: 2, 9: 4, 10: 8, 11: 4, 12: 8, 13: 4,
}
