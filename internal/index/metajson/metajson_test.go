package metajson_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/index/metajson"
)

// What front matter held comes back as the same Go types, however deep, so
// a template formats a date and compares a count the same in a list as on
// the item's own page.
func TestValuesComeBackAsTheTypesTheyWere(t *testing.T) {
	day := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	at := time.Date(2026, 3, 4, 9, 30, 0, 0, time.FixedZone("", 8*3600))
	meta := map[string]any{
		"day":    day,
		"events": []any{map[string]any{"at": at, "what": "talk"}},
		"weight": 3,
		"big":    int64(1) << 40,
		"rating": 4.0,
		"share":  0.25,
		"tiny":   1e-9,
		"text":   "2026-03-04",
		"yes":    true,
		"none":   nil,
		"empty":  []any{},
	}
	data, err := metajson.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got, err := metajson.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, data)
	}

	want := map[string]any{
		"day":    day,
		"events": []any{map[string]any{"at": at, "what": "talk"}},
		"weight": 3,
		"big":    1 << 40,
		"rating": 4.0,
		"share":  0.25,
		"tiny":   1e-9,
		"text":   "2026-03-04",
		"yes":    true,
		"none":   nil,
		"empty":  []any{},
	}
	for k, w := range want {
		g := got[k]
		if fmt.Sprintf("%T", g) != fmt.Sprintf("%T", w) {
			t.Errorf("%s: got a %T, want a %T (%s)", k, g, w, data)
			continue
		}
		if gt, ok := g.(time.Time); ok {
			if !gt.Equal(w.(time.Time)) || gt.Format(time.RFC3339) != w.(time.Time).Format(time.RFC3339) {
				t.Errorf("%s: got %v, want %v", k, gt, w)
			}
			continue
		}
		if k == "events" {
			ev := g.([]any)[0].(map[string]any)
			if _, ok := ev["at"].(time.Time); !ok || ev["what"] != "talk" {
				t.Errorf("events: got %#v", ev)
			}
			continue
		}
		if !reflect.DeepEqual(g, w) {
			t.Errorf("%s: got %#v, want %#v", k, g, w)
		}
	}
}

// Keys are written in order, so the same meta is the same bytes and the
// index rebuilt from scratch holds the same rows.
func TestTheSameMetaIsTheSameBytes(t *testing.T) {
	meta := map[string]any{"b": 1, "a": []any{2.0, "x"}, "c": map[string]any{"z": true, "y": nil}}
	first, err := metajson.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"a":[2.0,"x"],"b":1,"c":{"y":null,"z":true}}`; string(first) != want {
		t.Errorf("got %s, want %s", first, want)
	}
	for range 20 {
		again, _ := metajson.Marshal(meta)
		if string(again) != string(first) {
			t.Fatalf("%s then %s", first, again)
		}
	}
}
