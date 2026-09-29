package render

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/kite-plus/kite/internal/render/img"
	"github.com/kite-plus/kite/internal/render/url"
)

// Resource is a file a page's bundle holds, such as a picture beside its
// text, published beside the page.
type Resource interface {
	// Name is the file's path within the bundle, as images/01.jpg.
	Name() string
	RelPermalink() string
	Permalink() string
	// MediaType is the file's type, as image/jpeg.
	MediaType() string
	Content() (string, error)
	// Data holds a picture's Width and Height.
	Data() (map[string]any, error)
}

// ResourceList is the files of a page's bundle, in name order.
type ResourceList []Resource

// Get is the file of a name, nil when the bundle holds none.
func (l ResourceList) Get(name string) Resource {
	for _, r := range l {
		if r.Name() == name {
			return r
		}
	}
	return nil
}

// Match lists the files whose names match a pattern, in any case: "*.jpg"
// for those at the top of the bundle, "images/*" for those in a folder, and
// "**.jpg" for those anywhere in it, since * stops at a / and ** does not.
func (l ResourceList) Match(pattern string) ResourceList {
	pattern = strings.ToLower(pattern)
	match := func(name string) bool { ok, _ := path.Match(pattern, name); return ok }
	if strings.Contains(pattern, "**") {
		var b strings.Builder
		b.WriteString("^")
		for i, part := range strings.Split(pattern, "**") {
			if i > 0 {
				b.WriteString(".*")
			}
			for _, r := range part {
				switch r {
				case '*':
					b.WriteString("[^/]*")
				case '?':
					b.WriteString("[^/]")
				default:
					b.WriteString(regexp.QuoteMeta(string(r)))
				}
			}
		}
		b.WriteString("$")
		re := regexp.MustCompile(b.String())
		match = re.MatchString
	}
	var out ResourceList
	for _, r := range l {
		if match(strings.ToLower(r.Name())) {
			out = append(out, r)
		}
	}
	return out
}

// ByType lists the files of a type: image, video, audio, text or
// application.
func (l ResourceList) ByType(kind string) ResourceList {
	var out ResourceList
	for _, r := range l {
		if main, _, _ := strings.Cut(r.MediaType(), "/"); main == kind {
			out = append(out, r)
		}
	}
	return out
}

// Bundle is where a page's files are read and published.
type Bundle struct {
	// Media is rooted at the project, and Dir is the bundle's folder in it.
	Media fs.FS
	Dir   string

	// Out is the folder the page is published in, where its files go too.
	Out   string
	Links *url.Resolver

	// Images makes pictures from the bundle's, and Made is told where each
	// one is published. Without Images no picture is made.
	Images *img.Processor
	Made   func(out string, m *img.Made)
}

// Files returns the resources of a bundle for the files it publishes, named
// by their paths within it.
func (b Bundle) Files(names []string) ResourceList {
	out := make(ResourceList, 0, len(names))
	for _, name := range names {
		out = append(out, &file{bundle: b, name: name})
	}
	return out
}

type file struct {
	bundle Bundle
	name   string

	once sync.Once
	info img.Info
	err  error
}

func (f *file) Name() string      { return f.name }
func (f *file) MediaType() string { return mediaType(f.name) }
func (f *file) source() string    { return path.Join(f.bundle.Dir, f.name) }
func (f *file) out() string       { return path.Join(f.bundle.Out, f.name) }

func (f *file) RelPermalink() string {
	if f.bundle.Links == nil {
		return "/" + f.out()
	}
	return f.bundle.Links.Rel(f.out())
}

func (f *file) Permalink() string {
	if f.bundle.Links == nil {
		return f.RelPermalink()
	}
	return f.bundle.Links.Absolute(f.RelPermalink())
}

func (f *file) read() ([]byte, error) { return fs.ReadFile(f.bundle.Media, f.source()) }

func (f *file) Content() (string, error) {
	data, err := f.read()
	return string(data), err
}

func (f *file) Data() (map[string]any, error) {
	if !f.picture() {
		return map[string]any{}, nil
	}
	w, err := f.Width()
	if err != nil {
		return nil, err
	}
	h, _ := f.Height()
	return map[string]any{"Width": w, "Height": h}, nil
}

// Width and Height are a picture's, as it is shown, read from its header.
func (f *file) Width() (int, error) {
	info, err := f.inspect()
	return info.Width, err
}

func (f *file) Height() (int, error) {
	info, err := f.inspect()
	return info.Height, err
}

func (f *file) inspect() (img.Info, error) {
	f.once.Do(func() {
		if !f.picture() {
			f.err = fmt.Errorf("%s is not a picture", f.name)
			return
		}
		data, err := f.read()
		if err != nil {
			f.err = err
			return
		}
		f.info, f.err = img.Inspect(data)
	})
	return f.info, f.err
}

func (f *file) picture() bool {
	switch f.MediaType() {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	}
	return false
}

// Picture is a picture the img functions can make another from: a file of
// a bundle, or one made from it.
type Picture interface {
	Make(img.Recipe) (*Image, error)
	Recipe() img.Recipe
}

func (f *file) Recipe() img.Recipe { return img.Recipe{} }

func (f *file) Make(r img.Recipe) (*Image, error) {
	switch {
	case !f.picture():
		return nil, fmt.Errorf("%s is not a picture Kite can make another from", f.name)
	case f.bundle.Images == nil:
		return nil, fmt.Errorf("pictures cannot be made here")
	}
	return &Image{src: f, recipe: r}, nil
}

// Image is a picture made from a file of a bundle by a recipe. It is made
// the first time a template asks where it is or how large it is, and is
// published beside the file it was made from, named by what went into it.
type Image struct {
	src    *file
	recipe img.Recipe

	once sync.Once
	made *img.Made
	out  string
	err  error
}

func (i *Image) Recipe() img.Recipe { return i.recipe }

func (i *Image) Make(r img.Recipe) (*Image, error) { return i.src.Make(r) }

func (i *Image) make() error {
	i.once.Do(func() {
		src := i.src
		info, err := fs.Stat(src.bundle.Media, src.source())
		if err != nil {
			i.err = err
			return
		}
		hash, err := src.bundle.Images.Hash(src.source(), info.Size(), info.ModTime(), src.read)
		if err != nil {
			i.err = err
			return
		}
		if i.made, i.err = src.bundle.Images.Make(hash, src.read, i.recipe); i.err != nil {
			i.err = fmt.Errorf("%s: %w", src.name, i.err)
			return
		}
		base := strings.TrimSuffix(src.name, path.Ext(src.name))
		i.out = path.Join(src.bundle.Out, img.PublishedName(base, i.made))
		if src.bundle.Made != nil {
			src.bundle.Made(i.out, i.made)
		}
	})
	return i.err
}

// Name is the picture's name within the bundle, as it is published.
func (i *Image) Name() (string, error) {
	if err := i.make(); err != nil {
		return "", err
	}
	return strings.TrimPrefix(i.out, i.src.bundle.Out+"/"), nil
}

func (i *Image) RelPermalink() (string, error) {
	if err := i.make(); err != nil {
		return "", err
	}
	if i.src.bundle.Links == nil {
		return "/" + i.out, nil
	}
	return i.src.bundle.Links.Rel(i.out), nil
}

func (i *Image) Permalink() (string, error) {
	rel, err := i.RelPermalink()
	if err != nil || i.src.bundle.Links == nil {
		return rel, err
	}
	return i.src.bundle.Links.Absolute(rel), nil
}

func (i *Image) MediaType() (string, error) {
	if err := i.make(); err != nil {
		return "", err
	}
	return i.made.MediaType(), nil
}

func (i *Image) Width() (int, error) {
	if err := i.make(); err != nil {
		return 0, err
	}
	return i.made.Width, nil
}

func (i *Image) Height() (int, error) {
	if err := i.make(); err != nil {
		return 0, err
	}
	return i.made.Height, nil
}

func (i *Image) Data() (map[string]any, error) {
	if err := i.make(); err != nil {
		return nil, err
	}
	return map[string]any{"Width": i.made.Width, "Height": i.made.Height}, nil
}

func (i *Image) Content() (string, error) {
	if err := i.make(); err != nil {
		return "", err
	}
	data, err := i.made.Bytes()
	return string(data), err
}

// mediaType names a file's type by its extension, from a table of its own
// rather than the system's, which differs from one machine to the next.
func mediaType(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	case ".svg":
		return "image/svg+xml"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".txt":
		return "text/plain"
	case ".csv":
		return "text/csv"
	case ".json":
		return "application/json"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	}
	return "application/octet-stream"
}
