package api_test

import (
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/api"
)

// spec is the served description, decoded loosely so that the test reads the
// document a client would actually receive rather than the Go value it was
// built from.
type spec struct {
	OpenAPI string `json:"openapi"`
	Info    struct {
		Title string `json:"title"`
	} `json:"info"`
	Paths map[string]map[string]struct {
		OperationID string `json:"operationId"`
		Parameters  []struct {
			Name string `json:"name"`
			In   string `json:"in"`
		} `json:"parameters"`
		Responses map[string]struct {
			Content map[string]struct {
				Schema map[string]any `json:"schema"`
			} `json:"content"`
		} `json:"responses"`
	} `json:"paths"`
	Components struct {
		Schemas map[string]struct {
			Type       string                    `json:"type"`
			Properties map[string]map[string]any `json:"properties"`
			Required   []string                  `json:"required"`
		} `json:"schemas"`
	} `json:"components"`
}

func fetchSpec(t *testing.T, h http.Handler) spec {
	t.Helper()
	return get[spec](t, h, api.Prefix+api.OpenAPIPath, http.StatusOK)
}

// A description nobody checks is a description that drifts. This is the test
// that makes the document a contract rather than documentation: an endpoint
// cannot be served without being described.
func TestEveryServedEndpointIsDescribed(t *testing.T) {
	h, _ := newServer(t, newProject(t, 1))
	doc := fetchSpec(t, h)

	if doc.OpenAPI != "3.1.0" {
		t.Errorf("openapi = %q, want 3.1.0", doc.OpenAPI)
	}

	described := map[string]bool{}
	for path, methods := range doc.Paths {
		for method := range methods {
			described[strings.ToUpper(method)+" "+path] = true
		}
	}

	srv := api.New(api.Options{Site: func() api.View { return api.View{} }})
	for _, rt := range srv.Routes() {
		if rt == http.MethodGet+" "+api.OpenAPIPath {
			continue // the document does not describe itself
		}
		// A wildcard that takes the rest of a path is described as a plain
		// parameter, which is the only way OpenAPI has to spell it.
		rt = strings.ReplaceAll(rt, "...}", "}")
		if !described[rt] {
			t.Errorf("%s is served but not described", rt)
		}
	}
	if len(described) == 0 {
		t.Fatal("the document describes nothing")
	}
}

// The schemas are derived from the wire types, so a field added to one has to
// appear here without anybody writing it down.
func TestSchemasAreDerivedFromTheWireTypes(t *testing.T) {
	h, _ := newServer(t, newProject(t, 1))
	doc := fetchSpec(t, h)

	summary, ok := doc.Components.Schemas["Summary"]
	if !ok {
		t.Fatalf("no Summary schema; have %v", keysOf(doc.Components.Schemas))
	}
	for _, field := range []string{"id", "kind", "slug", "title", "status", "url", "revision", "created_at"} {
		if _, present := summary.Properties[field]; !present {
			t.Errorf("Summary schema is missing %q", field)
		}
	}
	if summary.Properties["created_at"]["format"] != "date-time" {
		t.Errorf("created_at format = %v, want date-time", summary.Properties["created_at"]["format"])
	}
	// An optional field must not be required, or a generated client will
	// insist on a value the server omits.
	if slices.Contains(summary.Required, "published_at") {
		t.Error("published_at is required although the server omits it when absent")
	}
	if !slices.Contains(summary.Required, "id") {
		t.Error("id is not required although it is always present")
	}

	// A generic instantiation is named the way a client would want to read it.
	if _, ok := doc.Components.Schemas["SummaryList"]; !ok {
		t.Errorf("no SummaryList schema; have %v", keysOf(doc.Components.Schemas))
	}

	// An item carries the body; the row in a listing does not.
	item := doc.Components.Schemas["Item"]
	if _, present := item.Properties["body"]; !present {
		t.Error("Item schema has no body")
	}
	if _, present := summary.Properties["body"]; present {
		t.Error("Summary schema carries a body, which a listing does not return")
	}
}

func TestFailureResponsesAreDescribed(t *testing.T) {
	h, _ := newServer(t, newProject(t, 1))
	doc := fetchSpec(t, h)

	for path, want := range map[string]string{
		"/contents":      "400",
		"/contents/{id}": "404",
	} {
		if _, ok := doc.Paths[path]["get"].Responses[want]; !ok {
			t.Errorf("GET %s does not describe a %s response", path, want)
		}
	}
}

func keysOf[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
