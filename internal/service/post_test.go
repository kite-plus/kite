package service

import (
	"testing"
	"time"

	"github.com/amigoer/kite-blog/internal/model"
)

func TestValidatePost(t *testing.T) {
	tests := []struct {
		name    string
		post    *model.Post
		wantErr bool
	}{
		{
			name:    "nil post",
			post:    nil,
			wantErr: true,
		},
		{
			name:    "empty title",
			post:    &model.Post{Slug: "test", ContentMarkdown: "md", ContentHTML: "<p>html</p>", Status: "draft"},
			wantErr: true,
		},
		{
			name:    "empty slug",
			post:    &model.Post{Title: "Test", ContentMarkdown: "md", ContentHTML: "<p>html</p>", Status: "draft"},
			wantErr: true,
		},
		{
			name:    "empty content_markdown",
			post:    &model.Post{Title: "Test", Slug: "test", ContentHTML: "<p>html</p>", Status: "draft"},
			wantErr: true,
		},
		{
			name:    "empty content_html",
			post:    &model.Post{Title: "Test", Slug: "test", ContentMarkdown: "md", Status: "draft"},
			wantErr: true,
		},
		{
			name:    "invalid status",
			post:    &model.Post{Title: "Test", Slug: "test", ContentMarkdown: "md", ContentHTML: "<p>html</p>", Status: "unknown"},
			wantErr: true,
		},
		{
			name:    "valid draft",
			post:    &model.Post{Title: "Test", Slug: "test", ContentMarkdown: "md", ContentHTML: "<p>html</p>", Status: "draft"},
			wantErr: false,
		},
		{
			name:    "valid published",
			post:    &model.Post{Title: "Test", Slug: "test", ContentMarkdown: "md", ContentHTML: "<p>html</p>", Status: "published"},
			wantErr: false,
		},
		{
			name:    "valid archived",
			post:    &model.Post{Title: "Test", Slug: "test", ContentMarkdown: "md", ContentHTML: "<p>html</p>", Status: "archived"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePost(tt.post)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", model.PostStatusDraft},
		{"  ", model.PostStatusDraft},
		{"draft", "draft"},
		{"published", "published"},
		{"archived", "archived"},
		{" published ", "published"},
	}

	for _, tt := range tests {
		got := normalizeStatus(tt.input)
		if got != tt.want {
			t.Errorf("normalizeStatus(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsValidStatus(t *testing.T) {
	valid := []string{"draft", "published", "archived"}
	invalid := []string{"", "unknown", "DRAFT", "pending"}

	for _, s := range valid {
		if !isValidStatus(s) {
			t.Errorf("isValidStatus(%q) should be true", s)
		}
	}
	for _, s := range invalid {
		if isValidStatus(s) {
			t.Errorf("isValidStatus(%q) should be false", s)
		}
	}
}

func TestNormalizeShowComments(t *testing.T) {
	boolTrue := true
	boolFalse := false

	if normalizeShowComments(nil, true) != true {
		t.Error("nil with fallback true should return true")
	}
	if normalizeShowComments(nil, false) != false {
		t.Error("nil with fallback false should return false")
	}
	if normalizeShowComments(&boolTrue, false) != true {
		t.Error("&true with fallback false should return true")
	}
	if normalizeShowComments(&boolFalse, true) != false {
		t.Error("&false with fallback true should return false")
	}
}

func TestNormalizeTimePointer(t *testing.T) {
	if normalizeTimePointer(nil) != nil {
		t.Error("nil should return nil")
	}

	loc, _ := time.LoadLocation("Asia/Shanghai")
	local := time.Date(2026, 3, 15, 10, 0, 0, 0, loc)
	result := normalizeTimePointer(&local)
	if result == nil || result.Location() != time.UTC {
		t.Error("should normalize to UTC")
	}
}

func TestParseUUID(t *testing.T) {
	_, err := parseUUID("")
	if err == nil {
		t.Error("empty should error")
	}

	_, err = parseUUID("not-a-uuid")
	if err == nil {
		t.Error("invalid uuid should error")
	}

	_, err = parseUUID("0195f3ff-4f17-7f0b-9e5f-15db2f2fb6a1")
	if err != nil {
		t.Errorf("valid uuid should not error: %v", err)
	}
}

func TestParseOptionalUUID(t *testing.T) {
	result, err := parseOptionalUUID("")
	if err != nil || result != nil {
		t.Error("empty should return nil, nil")
	}

	result, err = parseOptionalUUID("  ")
	if err != nil || result != nil {
		t.Error("whitespace should return nil, nil")
	}

	_, err = parseOptionalUUID("invalid")
	if err == nil {
		t.Error("invalid should error")
	}

	result, err = parseOptionalUUID("0195f3ff-4f17-7f0b-9e5f-15db2f2fb6a1")
	if err != nil || result == nil {
		t.Errorf("valid uuid should return parsed: err=%v", err)
	}
}
