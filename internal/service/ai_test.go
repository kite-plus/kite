package service

import (
	"testing"

	"github.com/amigoer/kite-blog/internal/config"
)

func TestNewAIService(t *testing.T) {
	cfg := &config.AIConfig{Enabled: false}
	svc := NewAIService(cfg)
	if svc == nil {
		t.Fatal("NewAIService should not return nil")
	}
}

func TestGenerateSummary_Disabled(t *testing.T) {
	svc := NewAIService(&config.AIConfig{Enabled: false})
	_, err := svc.GenerateSummary(SummaryInput{Content: "test"})
	if err == nil {
		t.Error("should error when AI is disabled")
	}
}

func TestSuggestTags_Disabled(t *testing.T) {
	svc := NewAIService(&config.AIConfig{Enabled: false})
	_, err := svc.SuggestTags(TagSuggestionInput{Title: "test", Content: "test"})
	if err == nil {
		t.Error("should error when AI is disabled")
	}
}
