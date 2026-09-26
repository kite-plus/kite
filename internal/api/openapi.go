package api

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/kite-plus/kite/internal/auth"
	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/publish"
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

	// Security applies to every operation that does not override it, which
	// is how "everything needs a session except the way in" is said once.
	Security []map[string][]string `json:"security,omitempty"`
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
	Schemas         map[string]*jsonSchema    `json:"schemas"`
	SecuritySchemes map[string]securityScheme `json:"securitySchemes,omitempty"`
}

// securityScheme describes how a caller proves who it is.
type securityScheme struct {
	Type        string `json:"type"`
	In          string `json:"in,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type pathItem struct {
	Get    *operation `json:"get,omitempty"`
	Post   *operation `json:"post,omitempty"`
	Put    *operation `json:"put,omitempty"`
	Delete *operation `json:"delete,omitempty"`
}

type operation struct {
	OperationID string              `json:"operationId"`
	Summary     string              `json:"summary"`
	Parameters  []parameter         `json:"parameters,omitempty"`
	RequestBody *requestBody        `json:"requestBody,omitempty"`
	Responses   map[string]response `json:"responses"`

	// Security overrides the document's. An empty list -- not an absent one
	// -- is how OpenAPI says "this one answers without a session".
	Security *[]map[string][]string `json:"security,omitempty"`
}

type requestBody struct {
	Required bool                 `json:"required"`
	Content  map[string]mediaType `json:"content"`
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
	"deleted_only":    {Desc: "Return only soft deleted items."},
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
	conflictRef := ref(ConflictBody{})

	jsonOf := func(schema *jsonSchema) map[string]mediaType {
		return map[string]mediaType{"application/json": {Schema: schema}}
	}
	body := func(schema *jsonSchema) *requestBody {
		return &requestBody{Required: true, Content: jsonOf(schema)}
	}
	pathParam := func(name string) parameter {
		return parameter{Name: name, In: "path", Required: true, Schema: &jsonSchema{Type: "string"}}
	}
	fails := func(codes ...string) map[string]response {
		out := map[string]response{}
		for _, code := range codes {
			out[code] = response{Description: "Failed.", Content: jsonOf(errorRef)}
		}
		return out
	}
	ok := func(schema *jsonSchema, desc string, failures ...string) map[string]response {
		out := fails(failures...)
		out["200"] = response{Description: desc, Content: jsonOf(schema)}
		return out
	}

	created := func(schema *jsonSchema, desc string, failures ...string) map[string]response {
		out := fails(failures...)
		out["201"] = response{Description: desc, Content: jsonOf(schema)}
		return out
	}
	// A refused publish carries the plan, so every reason can be shown at
	// once, and what did happen, since a push can fail after its commit.
	withRefused := func(schema *jsonSchema, out map[string]response) map[string]response {
		out["409"] = response{Description: "The publish did not fully happen.", Content: jsonOf(schema)}
		return out
	}
	// A change that asks for the current password is refused with 403 when
	// it is wrong: a 401 would tell the client its session had ended.
	wrongPassword := func(out map[string]response) map[string]response {
		out["403"] = response{
			Description: "The current password is wrong, or the request came from another site.",
			Content:     jsonOf(errorRef),
		}
		return out
	}
	// public marks an endpoint that answers without a session. It is a
	// variable rather than a literal because OpenAPI needs the empty list to
	// be present, and a pointer to it is the only way to say that in Go.
	public := []map[string][]string{}

	// A conflict is its own body, not the ordinary error shape: it carries the
	// version that is now stored so the client can show both.
	withConflict := func(schema *jsonSchema, out map[string]response) map[string]response {
		out["409"] = response{Description: "The item changed since it was loaded.", Content: jsonOf(schema)}
		return out
	}

	doc := &document{
		OpenAPI: "3.1.0",
		Info: info{
			Title:   "Kite read model",
			Version: buildinfo.Version,
			Description: "The read side of a Kite project. " +
				"Listings are paged by opaque cursor; there is no offset. " +
				"When the server has an account configured, every endpoint " +
				"but setup, sign-in and this description needs a session. " +
				"A server that has not been set up yet answers nothing else " +
				"at all until it has.",
		},
		Servers:  []server{{URL: Prefix}},
		Security: []map[string][]string{{"session": {}}},
		Paths: map[string]pathItem{
			"/setup": {
				Get: &operation{
					OperationID: "getSetupState",
					Summary:     "Report whether this server still has to be set up.",
					Security:    &public,
					Responses:   ok(ref(SetupState{}), "Whether setup is needed, and what to fill the form with."),
				},
				Post: &operation{
					OperationID: "setUp",
					Summary:     "Describe the site and create the account that guards it.",
					Security:    &public,
					RequestBody: body(ref(SetupRequest{})),
					Responses: ok(ref(SessionInfo{}),
						"Set up, and signed in. The cookie is in Set-Cookie.",
						"400", "405", "409"),
				},
			},
			"/auth/session": {Get: &operation{
				OperationID: "getSession",
				Summary:     "Report whether this server needs a sign-in, and whether the caller has one.",
				Security:    &public,
				Responses:   ok(ref(SessionInfo{}), "Whether a session is needed, and the current one."),
			}},
			"/auth/login": {Post: &operation{
				OperationID: "login",
				Summary:     "Exchange credentials for a session cookie.",
				Security:    &public,
				RequestBody: body(ref(Credentials{})),
				Responses: ok(ref(SessionInfo{}),
					"The session. The cookie is in Set-Cookie.", "400", "401", "429"),
			}},
			"/auth/logout": {Post: &operation{
				OperationID: "logout",
				Summary:     "Discard the session cookie.",
				Security:    &public,
				Responses:   map[string]response{"204": {Description: "Signed out."}},
			}},
			"/account": {Get: &operation{
				OperationID: "getAccount",
				Summary:     "Describe the person using the studio and how they sign in.",
				Responses:   ok(ref(AccountInfo{}), "The account and profile."),
			}},
			"/account/profile": {Put: &operation{
				OperationID: "updateProfile",
				Summary: "Change the name and email address the studio shows. They are kept " +
					"in .kite/secrets, never published.",
				RequestBody: body(ref(Profile{})),
				Responses:   ok(ref(AccountInfo{}), "The account as it now is.", "400", "405", "501"),
			}},
			"/account/avatar": {
				Get: &operation{
					OperationID: "getAvatar",
					Summary:     "Read the picture shown beside the name.",
					Responses: map[string]response{
						"200": {
							Description: "The picture.",
							Content:     map[string]mediaType{"image/*": {Schema: &jsonSchema{Type: "string", Format: "binary"}}},
						},
						"404": {Description: "Failed.", Content: jsonOf(errorRef)},
					},
				},
				Put: &operation{
					OperationID: "uploadAvatar",
					Summary:     "Choose the picture: PNG, JPEG, GIF or WebP, up to 1 MB.",
					RequestBody: &requestBody{
						Required: true,
						Content: map[string]mediaType{"multipart/form-data": {Schema: &jsonSchema{
							Type: "object",
							Properties: map[string]*jsonSchema{
								"file": {Type: "string", Format: "binary"},
							},
							Required: []string{"file"},
						}}},
					},
					Responses: ok(ref(AccountInfo{}), "The account as it now is.", "400", "405", "413", "415", "501"),
				},
				Delete: &operation{
					OperationID: "removeAvatar",
					Summary:     "Go back to initials.",
					Responses:   ok(ref(AccountInfo{}), "The account as it now is.", "405", "501"),
				},
			},
			"/account/credentials": {
				Put: &operation{
					OperationID: "setCredentials",
					Summary: "Give a studio with no password one, or change the name and password " +
						"of a guarded one. A change ends every other session; the caller's is issued " +
						"again in Set-Cookie.",
					RequestBody: body(ref(CredentialsChange{})),
					Responses: wrongPassword(ok(ref(AccountInfo{}), "The account as it now is.",
						"400", "405", "409", "429", "501")),
				},
				Delete: &operation{
					OperationID: "removeCredentials",
					Summary: "Take the password away, leaving the studio open. Only a server that " +
						"listens on this machine alone allows it.",
					RequestBody: body(ref(PasswordConfirmation{})),
					Responses: wrongPassword(ok(ref(AccountInfo{}), "The account as it now is.",
						"400", "405", "409", "429", "501")),
				},
			},
			"/account/sessions": {Delete: &operation{
				OperationID: "endOtherSessions",
				Summary: "Sign out every browser but the caller's, whose session is issued again " +
					"in Set-Cookie.",
				Responses: ok(ref(AccountInfo{}), "The account as it now is.", "400", "405", "409", "501"),
			}},
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
			"/publish": {
				Get: &operation{
					OperationID: "getDeliveryState",
					Summary:     "Report how far the content has traveled.",
					Responses:   ok(ref(publish.DeliveryState{}), "The delivery state."),
				},
				Post: &operation{
					OperationID: "publish",
					Summary:     "Commit the named content, and push it when asked to.",
					RequestBody: body(ref(PublishBody{})),
					Responses: withRefused(ref(PublishRefused{}),
						ok(ref(publish.Result{}), "What was published.", "400", "404", "405", "501")),
				},
			},
			"/publish/preflight": {Post: &operation{
				OperationID: "preflightPublish",
				Summary:     "Report what a publish would do, changing nothing.",
				RequestBody: body(ref(PublishBody{})),
				Responses:   ok(ref(publish.Plan{}), "The plan, with everything wrong with it.", "400", "404", "405", "501"),
			}},
			"/publish/push": {Post: &operation{
				OperationID: "push",
				Summary:     "Push what is already committed, replaying it onto a remote that moved on when asked to.",
				RequestBody: body(ref(PushBody{})),
				Responses: withRefused(ref(PublishRefused{}),
					ok(ref(publish.Result{}), "What was pushed.", "400", "405", "501")),
			}},
			"/export": {Post: &operation{
				OperationID: "exportSite",
				Summary: "Build the site as a deployment gets it, without drafts, and answer with it " +
					"as a zip archive of what kite build writes, to upload anywhere that serves static files.",
				Responses: map[string]response{
					"200": {
						Description: "The site. Content-Disposition names the archive.",
						Content:     map[string]mediaType{"application/zip": {Schema: &jsonSchema{Type: "string", Format: "binary"}}},
					},
					"409": {Description: "The site could not be built as it stands.", Content: jsonOf(errorRef)},
					"501": {Description: "Failed.", Content: jsonOf(errorRef)},
				},
			}},
			"/settings": {
				Get: &operation{
					OperationID: "getSettings",
					Summary:     "Read what can be configured, and what it is set to.",
					Responses:   ok(ref(Settings{}), "The settings. ETag carries the revision of kite.yaml."),
				},
				Put: &operation{
					OperationID: "updateSettings",
					Summary:     "Change configuration values by dotted path.",
					Parameters:  []parameter{ifMatch(true)},
					RequestBody: &requestBody{
						Required: true,
						Content: map[string]mediaType{"application/json": {Schema: &jsonSchema{
							Type:                 "object",
							Description:          "Dotted paths to values, such as site.title.",
							AdditionalProperties: &jsonSchema{},
						}}},
					},
					Responses: ok(ref(Settings{}), "The settings as stored.", "400", "405", "409", "428"),
				},
			},
			"/contents": {
				Get: &operation{
					OperationID: "listContents",
					Summary:     "List content.",
					Parameters:  listParameters(),
					Responses:   ok(ref(List[Summary]{}), "One page of items.", "400"),
				},
				Post: &operation{
					OperationID: "createContent",
					Summary:     "Write a new item.",
					RequestBody: body(ref(Draft{})),
					Responses:   created(ref(Item{}), "The item as stored.", "400", "405"),
				},
			},
			"/contents/{id}": {
				Get: &operation{
					OperationID: "getContent",
					Summary:     "Read one item, including its source body.",
					Parameters:  []parameter{pathParam("id")},
					Responses:   ok(ref(Item{}), "The item. ETag carries its revision.", "404"),
				},
				Put: &operation{
					OperationID: "updateContent",
					Summary:     "Replace an item, refusing an edit made against a replaced version.",
					Parameters:  []parameter{pathParam("id"), ifMatch(true)},
					RequestBody: body(ref(Draft{})),
					Responses: withConflict(conflictRef,
						ok(ref(Item{}), "The item as stored.", "400", "404", "405", "428")),
				},
				Delete: &operation{
					OperationID: "deleteContent",
					Summary:     "Move an item to the trash, keeping its files so it can be restored.",
					Parameters:  []parameter{pathParam("id"), ifMatch(true)},
					Responses: withConflict(conflictRef,
						map[string]response{
							"204": {Description: "Moved to the trash."},
							"404": {Description: "Failed.", Content: jsonOf(errorRef)},
							"405": {Description: "Failed.", Content: jsonOf(errorRef)},
							"428": {Description: "Failed.", Content: jsonOf(errorRef)},
						}),
				},
			},
			"/contents/{id}/restore": {Post: &operation{
				OperationID: "restoreContent",
				Summary:     "Restore an item from the trash.",
				Parameters:  []parameter{pathParam("id"), ifMatch(true)},
				Responses: withConflict(conflictRef,
					ok(ref(Item{}), "The item as restored.", "400", "404", "405", "428")),
			}},
			"/preview": {Post: &operation{
				OperationID: "preview",
				Summary:     "Render a draft through the real theme, without storing it.",
				Parameters: []parameter{{
					Name: "id", In: "query",
					Description: "The item being edited, so its own images resolve.",
					Schema:      &jsonSchema{Type: "string"},
				}},
				RequestBody: body(ref(Draft{})),
				Responses: map[string]response{
					"200": {
						Description: "The rendered page.",
						Content:     map[string]mediaType{"text/html": {Schema: &jsonSchema{Type: "string"}}},
					},
					"400": {Description: "Failed.", Content: jsonOf(errorRef)},
					"501": {Description: "Failed.", Content: jsonOf(errorRef)},
				},
			}},
			"/wordcount": {Post: &operation{
				OperationID: "countWords",
				Summary: "Count a draft's words as its page will: paragraphs, list items, " +
					"headings and table cells, without code blocks or image alt text.",
				Parameters: []parameter{{
					Name: "id", In: "query",
					Description: "The item being edited, so it is counted where it lives.",
					Schema:      &jsonSchema{Type: "string"},
				}},
				RequestBody: body(ref(Draft{})),
				Responses:   ok(ref(WordCount{}), "The count.", "400", "501"),
			}},
			"/contents/{id}/media": {Post: &operation{
				OperationID: "uploadMedia",
				Summary:     "Store a file beside a page and report the link that reaches it.",
				Parameters:  []parameter{pathParam("id")},
				RequestBody: &requestBody{
					Required: true,
					Content: map[string]mediaType{"multipart/form-data": {Schema: &jsonSchema{
						Type: "object",
						Properties: map[string]*jsonSchema{
							"file": {Type: "string", Format: "binary"},
						},
						Required: []string{"file"},
					}}},
				},
				Responses: created(ref(Media{}),
					"The stored file. The name may differ from the one sent, since an "+
						"upload never replaces a file of the same name.",
					"400", "404", "405", "413", "415"),
			}},
			"/contents/{id}/media/{name}": {Delete: &operation{
				OperationID: "deleteMedia",
				Summary:     "Remove a file from a page's bundle.",
				Parameters:  []parameter{pathParam("id"), pathParam("name")},
				Responses: map[string]response{
					"204": {Description: "Removed."},
					"404": {Description: "Failed.", Content: jsonOf(errorRef)},
					"405": {Description: "Failed.", Content: jsonOf(errorRef)},
				},
			}},
			"/taxonomies": {Get: &operation{
				OperationID: "listTaxonomies",
				Summary:     "List taxonomies and how many terms each holds.",
				Responses:   ok(ref(List[Taxonomy]{}), "The taxonomies."),
			}},
			"/taxonomies/{taxonomy}/terms": {Get: &operation{
				OperationID: "listTerms",
				Summary:     "Count one taxonomy's terms over a filtered set.",
				Parameters:  append([]parameter{pathParam("taxonomy")}, listParameters()...),
				Responses:   ok(ref(List[TermCount]{}), "Term counts, most used first.", "400", "404"),
			}},
			"/taxonomies/{taxonomy}/terms/{term}": {
				Get: &operation{
					OperationID: "getTerm",
					Summary:     "Read one term with every item that carries it, trashed ones included.",
					Parameters:  []parameter{pathParam("taxonomy"), pathParam("term")},
					Responses: ok(ref(TermDetail{}),
						"The term and its items. ETag fingerprints them for a rename or removal.", "404"),
				},
				Put: &operation{
					OperationID: "renameTerm",
					Summary: "Rename a term on every item that carries it, merging it into " +
						"another term when the new name is already in use.",
					Parameters:  []parameter{pathParam("taxonomy"), pathParam("term"), ifMatch(true)},
					RequestBody: body(ref(TermRename{})),
					Responses: ok(ref(TermDetail{}), "The term under its new name.",
						"400", "404", "405", "409", "428"),
				},
				Delete: &operation{
					OperationID: "removeTerm",
					Summary:     "Take a term off every item that carries it. The items stay.",
					Parameters:  []parameter{pathParam("taxonomy"), pathParam("term"), ifMatch(true)},
					Responses: map[string]response{
						"204": {Description: "Taken off every item."},
						"404": {Description: "Failed.", Content: jsonOf(errorRef)},
						"405": {Description: "Failed.", Content: jsonOf(errorRef)},
						"409": {Description: "Failed.", Content: jsonOf(errorRef)},
						"428": {Description: "Failed.", Content: jsonOf(errorRef)},
					},
				},
			},
			"/media": {Post: &operation{
				OperationID: "uploadSiteMedia",
				Summary: "Store a file that belongs to the site rather than one page, such as " +
					"a logo a theme setting names, in static/uploads.",
				RequestBody: &requestBody{
					Required: true,
					Content: map[string]mediaType{"multipart/form-data": {Schema: &jsonSchema{
						Type: "object",
						Properties: map[string]*jsonSchema{
							"file": {Type: "string", Format: "binary"},
						},
						Required: []string{"file"},
					}}},
				},
				Responses: created(ref(Media{}),
					"The stored file. Link is its path in the site, which is what a setting holds.",
					"400", "405", "413", "415"),
			}},
			"/themes": {
				Get: &operation{
					OperationID: "listThemes",
					Summary: "List the themes the project could use: the one built into Kite, then " +
						"each directory of themes/, including the ones that cannot be used, with why. " +
						"What a theme says about itself is in the language Accept-Language asks for, " +
						"when the theme has a language pack for it.",
					Responses: ok(ref(List[ThemeInfo]{}), "The themes.", "501"),
				},
				Post: &operation{
					OperationID: "installTheme",
					Summary: "Install a theme from a zip archive holding theme.yaml at its top or in " +
						"one folder. It is checked the way a theme is checked when a site loads it.",
					Parameters: []parameter{{
						Name: "replace", In: "query",
						Description: "Replace an installed theme of the same name. Without it, one " +
							"already installed is answered with 409 and both versions.",
						Schema: &jsonSchema{Type: "boolean"},
					}},
					RequestBody: &requestBody{
						Required: true,
						Content: map[string]mediaType{"multipart/form-data": {Schema: &jsonSchema{
							Type: "object",
							Properties: map[string]*jsonSchema{
								"file": {Type: "string", Format: "binary"},
							},
							Required: []string{"file"},
						}}},
					},
					Responses: func() map[string]response {
						out := created(ref(ThemeInfo{}), "The installed theme.", "400", "405", "413", "501")
						out["409"] = response{
							Description: "A theme of that name is installed already.",
							Content:     jsonOf(ref(ThemeExists{})),
						}
						return out
					}(),
				},
			},
			"/themes/{name}": {
				Get: &operation{
					OperationID: "getTheme",
					Summary: "Describe a theme with what it can be configured with and what it would " +
						"read from kite.yaml. ETag carries the revision of kite.yaml, for saving.",
					Parameters: []parameter{pathParam("name")},
					Responses:  ok(ref(ThemeDetail{}), "The theme.", "404", "501"),
				},
				Delete: &operation{
					OperationID: "removeTheme",
					Summary:     "Remove an installed theme. The built-in theme and the one in use stay.",
					Parameters:  []parameter{pathParam("name")},
					Responses: map[string]response{
						"204": {Description: "Removed."},
						"400": {Description: "Failed.", Content: jsonOf(errorRef)},
						"404": {Description: "Failed.", Content: jsonOf(errorRef)},
						"405": {Description: "Failed.", Content: jsonOf(errorRef)},
						"409": {Description: "Failed.", Content: jsonOf(errorRef)},
					},
				},
			},
			"/themes/{name}/screenshot": {Get: &operation{
				OperationID: "getThemeScreenshot",
				Summary:     "Read a theme's picture of itself.",
				Parameters:  []parameter{pathParam("name")},
				Responses: map[string]response{
					"200": {
						Description: "The picture.",
						Content:     map[string]mediaType{"image/*": {Schema: &jsonSchema{Type: "string", Format: "binary"}}},
					},
					"404": {Description: "Failed.", Content: jsonOf(errorRef)},
				},
			}},
			"/previews": {Post: &operation{
				OperationID: "openPreview",
				Summary: "Draw the site with a theme or settings being tried, without writing " +
					"anything. The preview is kept while it is used.",
				RequestBody: body(ref(PreviewBody{})),
				Responses:   created(ref(Preview{}), "The open preview.", "400", "501"),
			}},
			"/previews/{token}": {
				Put: &operation{
					OperationID: "updatePreview",
					Summary:     "Draw an open preview again with another theme or other settings.",
					Parameters:  []parameter{pathParam("token")},
					RequestBody: body(ref(PreviewBody{})),
					Responses:   ok(ref(Preview{}), "The preview, at the same address.", "400", "404", "501"),
				},
				Delete: &operation{
					OperationID: "closePreview",
					Summary:     "Forget a preview.",
					Parameters:  []parameter{pathParam("token")},
					Responses:   map[string]response{"204": {Description: "Forgotten."}},
				},
			},
			"/previews/{token}/{path}": {Get: &operation{
				OperationID: "getPreviewPage",
				Summary: "Read a page or a file of a preview. Its links lead to its other pages, " +
					"drawn the same way.",
				Parameters: []parameter{pathParam("token"), {
					Name: "path", In: "path", Required: true,
					Description: "Where the page or file is in the site; it may hold slashes.",
					Schema:      &jsonSchema{Type: "string"},
				}},
				Responses: map[string]response{
					"200": {
						Description: "The page or file.",
						Content:     map[string]mediaType{"text/html": {Schema: &jsonSchema{Type: "string"}}},
					},
					"404": {Description: "Failed.", Content: jsonOf(errorRef)},
					"501": {Description: "Failed.", Content: jsonOf(errorRef)},
				},
			}},
		},
		Components: components{
			Schemas: schemas,
			SecuritySchemes: map[string]securityScheme{"session": {
				Type: "apiKey",
				In:   "cookie",
				Name: auth.CookieName,
				Description: "Set by POST /auth/login. A server with no account " +
					"configured accepts every request without one.",
			}},
		},
	}

	// Two refusals apply to nearly every endpoint, and writing them out
	// fifteen times above would bury what actually differs between them: a
	// session is needed unless the operation says otherwise, and anything
	// that changes something is refused when it came from another site.
	for _, item := range doc.Paths {
		for method, op := range map[string]*operation{
			http.MethodGet:    item.Get,
			http.MethodPost:   item.Post,
			http.MethodPut:    item.Put,
			http.MethodDelete: item.Delete,
		} {
			if op == nil {
				continue
			}
			if op.Security == nil {
				op.Responses["401"] = response{
					Description: "Not signed in.", Content: jsonOf(errorRef),
				}
			}
			if _, described := op.Responses["403"]; method != http.MethodGet && !described {
				op.Responses["403"] = response{
					Description: "The request came from another site.", Content: jsonOf(errorRef),
				}
			}
		}
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

// ifMatch describes the precondition every write to an existing item carries.
func ifMatch(required bool) parameter {
	return parameter{
		Name: "If-Match", In: "header", Required: required,
		Description: "The revision this edit was made against, as returned in ETag. " +
			"Required: without it a save would overwrite whatever is there.",
		Schema: &jsonSchema{Type: "string"},
	}
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
