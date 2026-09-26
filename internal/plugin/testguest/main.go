//go:build wasip1

// Command testguest is the plugin.wasm the plugin tests run, built by them
// with the Go toolchain. It exports every hook a plugin can, each doing a
// little of what a real plugin does, and its mode setting makes it misbehave
// for the tests that need it to.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/extism/go-pdk"
)

type settings struct {
	Sign string `json:"sign"`
	Mode string `json:"mode"`
}

type page struct {
	URL   string `json:"url"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Type  string `json:"type"`
	Text  string `json:"text"`
}

func main() {}

// misbehave does what the mode setting asks for, and reports whether that
// was to fail.
func misbehave(mode string) bool {
	switch mode {
	case "fail":
		pdk.SetError(errors.New("refused on purpose"))
		return true
	case "spin":
		for {
		}
	case "fetch":
		pdk.NewHTTPRequest(pdk.MethodGet, "https://example.com/").Send()
	}
	return false
}

//go:wasmexport transform_markdown
func transformMarkdown() int32 {
	var in struct {
		Settings settings `json:"settings"`
		Page     page     `json:"page"`
		Markdown string   `json:"markdown"`
	}
	if err := pdk.InputJSON(&in); err != nil {
		pdk.SetError(err)
		return 1
	}
	if misbehave(in.Settings.Mode) {
		return 1
	}
	if !strings.Contains(in.Markdown, ":kite:") {
		return 0
	}
	out := strings.ReplaceAll(in.Markdown, ":kite:", "a kite at "+in.Page.URL)
	if err := pdk.OutputJSON(map[string]string{"markdown": out}); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport transform_html
func transformHTML() int32 {
	var in struct {
		Settings settings `json:"settings"`
		Page     page     `json:"page"`
		HTML     string   `json:"html"`
	}
	if err := pdk.InputJSON(&in); err != nil {
		pdk.SetError(err)
		return 1
	}
	meta := fmt.Sprintf(`<meta name="signed" content="%s %s %s">`, in.Settings.Sign, in.Page.Kind, in.Page.URL)
	out := strings.Replace(in.HTML, "</head>", meta+"</head>", 1)
	if err := pdk.OutputJSON(map[string]string{"html": out}); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport build_complete
func buildComplete() int32 {
	var in struct {
		Settings settings `json:"settings"`
		Pages    []page   `json:"pages"`
	}
	if err := pdk.InputJSON(&in); err != nil {
		pdk.SetError(err)
		return 1
	}
	var index []page
	for _, p := range in.Pages {
		if p.Kind == "single" {
			index = append(index, p)
		}
	}
	data, err := json.Marshal(index)
	if err != nil {
		pdk.SetError(err)
		return 1
	}
	name := "index.json"
	if in.Settings.Mode == "escape" {
		name = "../../outside.json"
	}
	// Neither of these may differ between two builds of the same site.
	chance := fmt.Sprintf("%d %d", time.Now().UnixNano(), rand.Int64())
	files := []map[string]string{
		{"path": name, "content": string(data)},
		{"path": "chance.txt", "content": chance},
	}
	if err := pdk.OutputJSON(map[string]any{"files": files}); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}
