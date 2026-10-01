package apps

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

// harmlessSVG reports whether data is an SVG with no scripts, no event
// handlers, nothing embedded and nothing fetched from elsewhere: a picture
// and only that, as kite-plus/apps checks an icon before listing it. It is
// checked again here because the file is read from the package's repository,
// whose tag can be moved after it was listed.
func harmlessSVG(data []byte) bool {
	dec := xml.NewDecoder(bytes.NewReader(data))
	root, inStyle := "", false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return root == "svg"
		}
		if err != nil {
			return false
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if root == "" {
				if root = t.Name.Local; root != "svg" {
					return false
				}
			}
			switch strings.ToLower(t.Name.Local) {
			case "script", "foreignobject", "iframe", "object", "embed", "audio", "video":
				return false
			}
			inStyle = t.Name.Local == "style"
			for _, at := range t.Attr {
				name := strings.ToLower(at.Name.Local)
				value := strings.ToLower(strings.TrimSpace(at.Value))
				if strings.HasPrefix(name, "on") || fetchesCSS(value) ||
					name == "href" && !strings.HasPrefix(value, "#") && !strings.HasPrefix(value, "data:image/") {
					return false
				}
			}
		case xml.EndElement:
			inStyle = false
		case xml.CharData:
			if inStyle && fetchesCSS(string(t)) {
				return false
			}
		}
	}
}

// fetchesCSS reports whether CSS loads something from outside the picture:
// an @import, or a url() that is not a fragment of it.
func fetchesCSS(css string) bool {
	css = strings.ToLower(css)
	if strings.Contains(css, "@import") {
		return true
	}
	for rest := css; ; {
		i := strings.Index(rest, "url(")
		if i < 0 {
			return false
		}
		rest = strings.TrimLeft(rest[i+len("url("):], " \t\n'\"")
		if !strings.HasPrefix(rest, "#") {
			return true
		}
	}
}
