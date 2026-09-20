package content

import "github.com/oklog/ulid/v2"

// NewID mints a content ID. ULIDs are used rather than random UUIDs because
// they sort by creation time, which makes both cursor pagination and manual
// inspection of front matter easier.
func NewID() ID { return ID(ulid.Make().String()) }

// ValidID reports whether s is a syntactically valid content ID.
func ValidID(s string) bool {
	_, err := ulid.ParseStrict(s)
	return err == nil
}
