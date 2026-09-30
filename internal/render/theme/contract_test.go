package theme

import (
	"fmt"
	"html/template"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/render"
)

// frozen holds the theme contract kite/v1 as it was frozen: every method and
// function a template can call, with its signature.
const frozen = "testdata/kite-v1.txt"

// The contract kite/v1 only grows. A method or a function may be added, and
// then belongs in the frozen list and in theme-system.md, but one a theme may
// already call is never renamed, removed or given another signature: that is
// kite/v2.
func TestTheContractOnlyGrows(t *testing.T) {
	data, err := os.ReadFile(frozen)
	if err != nil {
		t.Fatal(err)
	}
	var was []string
	for _, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			was = append(was, line)
		}
	}
	now := contract()

	var gone, added []string
	for _, line := range was {
		if !slices.Contains(now, line) {
			gone = append(gone, line)
		}
	}
	for _, line := range now {
		if !slices.Contains(was, line) {
			added = append(added, line)
		}
	}
	if len(gone) > 0 {
		t.Errorf("kite/v1 is frozen, and a theme may call these as they were:\n  %s", strings.Join(gone, "\n  "))
	}
	if len(added) > 0 {
		t.Errorf("new to the contract; add them to %s and describe them in docs/design/theme-system.md:\n  %s",
			frozen, strings.Join(added, "\n  "))
	}
}

// contract lists what a template can call: the methods of what it is given,
// then the functions, namespaced ones by namespace, each with its signature.
func contract() []string {
	var out []string
	for _, typ := range []struct {
		name string
		of   reflect.Type
	}{
		{"Context", reflect.TypeFor[render.Context]()},
		{"Site", reflect.TypeFor[render.Site]()},
		{"Page", reflect.TypeFor[render.Page]()},
		{"Heading", reflect.TypeFor[render.Heading]()},
		{"Term", reflect.TypeFor[render.Term]()},
		{"MenuItem", reflect.TypeFor[render.MenuItem]()},
		{"Paginator", reflect.TypeFor[render.Paginator]()},
		{"Request", reflect.TypeFor[render.Request]()},
		{"Resource", reflect.TypeFor[render.Resource]()},
		{"ResourceList", reflect.TypeFor[render.ResourceList]()},
		{"Shortcode", reflect.TypeFor[render.Shortcode]()},
		{"BodyImage", reflect.TypeFor[render.BodyImage]()},
	} {
		out = append(out, methods(typ.name, typ.of)...)
	}
	// A picture of a page's bundle also has its size.
	picture := reflect.TypeOf(render.Bundle{}.Files([]string{"a.jpg"})[0])
	for _, name := range []string{"Width", "Height"} {
		m, _ := picture.MethodByName(name)
		out = append(out, "Resource(picture)."+name+" "+signature(m.Type, true))
	}
	// What img.* makes is a picture of its own. Make and Recipe are how the
	// img functions chain, not something a template calls.
	for _, line := range methods("Image", reflect.TypeFor[*render.Image]()) {
		if !strings.HasPrefix(line, "Image.Make ") && !strings.HasPrefix(line, "Image.Recipe ") {
			out = append(out, line)
		}
	}

	funcs := baseFuncs(nil, nil)
	funcs["partial"] = partialFunc(template.New("contract"))
	funcs["partialCached"] = partialCachedFunc(template.New("contract"))
	for _, name := range slices.Sorted(maps.Keys(funcs)) {
		fn := reflect.TypeOf(funcs[name])
		if ns := fn; ns.NumIn() == 0 && ns.NumOut() == 1 && ns.Out(0).Kind() == reflect.Struct {
			out = append(out, methods(name, ns.Out(0))...)
			continue
		}
		out = append(out, name+" "+signature(fn, false))
	}
	return out
}

// methods lists the exported methods of a type as name.Method signature.
func methods(name string, typ reflect.Type) []string {
	var out []string
	for i := range typ.NumMethod() {
		m := typ.Method(i)
		if m.IsExported() {
			out = append(out, fmt.Sprintf("%s.%s %s", name, m.Name, signature(m.Type, typ.Kind() != reflect.Interface)))
		}
	}
	return out
}

// signature writes a function's type, leaving out the receiver a method of a
// concrete type takes first.
func signature(fn reflect.Type, receiver bool) string {
	var in, out []string
	for i := range fn.NumIn() {
		if i == 0 && receiver {
			continue
		}
		arg := fn.In(i)
		if fn.IsVariadic() && i == fn.NumIn()-1 {
			in = append(in, "..."+arg.Elem().String())
			continue
		}
		in = append(in, arg.String())
	}
	for i := range fn.NumOut() {
		out = append(out, fn.Out(i).String())
	}
	s := "func(" + strings.Join(in, ", ") + ")"
	switch len(out) {
	case 0:
	case 1:
		s += " " + out[0]
	default:
		s += " (" + strings.Join(out, ", ") + ")"
	}
	return s
}
