package api

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/kite-plus/kite/internal/buildinfo"
)

// OpenAPIPath is where the description of this API is served.
const OpenAPIPath = "/openapi.json"

// document is an OpenAPI 3.1 description.
type document struct {
	OpenAPI string   `json:"openapi"`
	Info    info     `json:"info"`
	Servers []server `json:"servers"`

	// Paths are relative to the server entry, which is where the prefix
	// lives: a generated client should not have to be told it separately.
	Paths      map[string]pathItem `json:"paths"`
	Components components          `json:"components"`
}

type server struct {
	URL string `json:"url"`
}

type info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

type components struct {
	Schemas map[string]*jsonSchema `json:"schemas"`
}

type pathItem struct {
	Get *operation `json:"get,omitempty"`
}

type operation struct {
	OperationID string              `json:"operationId"`
	Summary     string              `json:"summary"`
	Parameters  []parameter         `json:"parameters,omitempty"`
	Responses   map[string]response `json:"responses"`
}

type parameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"`
	Required    bool        `json:"required,omitempty"`
	Description string      `json:"description,omitempty"`
	Explode     *bool       `json:"explode,omitempty"`
	Schema      *jsonSchema `json:"schema"`
}

type response struct {
	Description string               `json:"description"`
	Content     map[string]mediaType `json:"content,omitempty"`
}

type mediaType struct {
	Schema *jsonSchema `json:"schema"`
}

// paramDoc describes one listing parameter.
//
// The descriptions live next to the names so that [knownParams], which is what
// the parser actually enforces, stays the single list: a parameter the server
// accepts but nobody documented shows up as a hole here, and a documented one
// the server would refuse cannot exist.
var paramDoc = map[string]struct {
	Desc     string
	Repeated bool
}{
	"kind":            {Desc: "Content kind, repeatable.", Repeated: true},
	"status":          {Desc: "draft, scheduled, published or archived, repeatable.", Repeated: true},
	"locale":          {Desc: "Locale, repeatable.", Repeated: true},
	"id":              {Desc: "Restrict to these ids, repeatable.", Repeated: true},
	"term":            {Desc: "taxonomy:term the item must carry at least one of, repeatable.", Repeated: true},
	"term_all":        {Desc: "taxonomy:term the item must carry all of, repeatable.", Repeated: true},
	"published_from":  {Desc: "Earliest publication time, RFC 3339."},
	"published_to":    {Desc: "Latest publication time, RFC 3339."},
	"updated_from":    {Desc: "Earliest update time, RFC 3339."},
	"updated_to":      {Desc: "Latest update time, RFC 3339."},
	"q":               {Desc: "Free text over title, excerpt and body."},
	"include_deleted": {Desc: "Include soft deleted items."},
	"sort":            {Desc: "Comma separated keys, '-' for descending, e.g. -published_at,title."},
	"cursor":          {Desc: "Opaque position from a previous response. There is no offset."},
	"limit":           {Desc: "Items per page, capped by the server."},
	"count":           {Desc: "Also report the size of the whole filtered set."},
}

// openAPI builds the description of this API from the types that implement it.
func openAPI() *document {
	schemas := map[string]*jsonSchema{}
	ref := func(v any) *jsonSchema {
		return schemaFor(reflect.TypeOf(v), schemas)
	}

	errorRef := ref(ErrorBody{})
	fails := func(codes ...string) map[string]response {
		out := map[string]response{}
		for _, code := range codes {
			out[code] = response{
				Description: "Failed.",
				Content:     map[string]mediaType{"application/json": {Schema: errorRef}},
			}
		}
		return out
	}
	ok := func(schema *jsonSchema, desc string, failures ...string) map[string]response {
		out := fails(failures...)
		out["200"] = response{
			Description: desc,
			Content:     map[string]mediaType{"application/json": {Schema: schema}},
		}
		return out
	}

	doc := &document{
		OpenAPI: "3.1.0",
		Info: info{
			Title:   "Kite read model",
			Version: buildinfo.Version,
			Description: "The read side of a Kite project. " +
				"Listings are paged by opaque cursor; there is no offset.",
		},
		Servers: []server{{URL: Prefix}},
		Paths: map[string]pathItem{
			"/site": {Get: &operation{
				OperationID: "getSite",
				Summary:     "Describe the open project.",
				Responses:   ok(ref(SiteInfo{}), "The project."),
			}},
			"/content-types": {Get: &operation{
				OperationID: "listContentTypes",
				Summary:     "List content types and the field schema forms are generated from.",
				Responses:   ok(ref(List[ContentType]{}), "The registry."),
			}},
			"/contents": {Get: &operation{
				OperationID: "listContents",
				Summary:     "List content.",
				Parameters:  listParameters(),
				Responses:   ok(ref(List[Summary]{}), "One page of items.", "400"),
			}},
			"/contents/{id}": {Get: &operation{
				OperationID: "getContent",
				Summary:     "Read one item, including its source body.",
				Parameters: []parameter{{
					Name: "id", In: "path", Required: true,
					Schema: &jsonSchema{Type: "string"},
				}},
				Responses: ok(ref(Item{}), "The item.", "404"),
			}},
			"/taxonomies": {Get: &operation{
				OperationID: "listTaxonomies",
				Summary:     "List taxonomies and how many terms each holds.",
				Responses:   ok(ref(List[Taxonomy]{}), "The taxonomies."),
			}},
			"/taxonomies/{taxonomy}/terms": {Get: &operation{
				OperationID: "listTerms",
				Summary:     "Count one taxonomy's terms over a filtered set.",
				Parameters: append([]parameter{{
					Name: "taxonomy", In: "path", Required: true,
					Schema: &jsonSchema{Type: "string"},
				}}, listParameters()...),
				Responses: ok(ref(List[TermCount]{}), "Term counts, most used first.", "400", "404"),
			}},
		},
		Components: components{Schemas: schemas},
	}
	return doc
}

// listParameters describes every parameter a listing accepts, in the order
// [knownParams] declares them.
func listParameters() []parameter {
	explode := true

	out := make([]parameter, 0, len(knownParams))
	for _, name := range knownParams {
		doc := paramDoc[name]
		p := parameter{
			Name:        name,
			In:          "query",
			Description: doc.Desc,
			Schema:      &jsonSchema{Type: "string"},
		}
		if doc.Repeated {
			p.Schema = &jsonSchema{Type: "array", Items: &jsonSchema{Type: "string"}}
			p.Explode = &explode
		}
		switch name {
		case "limit":
			p.Schema = &jsonSchema{Type: "integer"}
		case "count", "include_deleted":
			p.Schema = &jsonSchema{Type: "boolean"}
		}
		out = append(out, p)
	}
	return out
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, openAPI())
}

// Document returns the API description as indented JSON.
//
// It is exported so the description can be produced without running a server,
// which is what lets a client be generated in a build that has no port to
// listen on.
func Document() ([]byte, error) {
	return json.MarshalIndent(openAPI(), "", "  ")
}
