package logger

import (
	"bytes"
	"log"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"DEBUG":   slog.LevelDebug,
		"info":    slog.LevelInfo,
		"warn":    slog.LevelWarn,
		"WARNING": slog.LevelWarn,
		"error":   slog.LevelError,
		"":        slog.LevelInfo,
		"other":   slog.LevelInfo,
		" warn ":  slog.LevelWarn,
	}
	for input, want := range cases {
		if got := ParseLevel(input); got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestParseFormat(t *testing.T) {
	cases := map[string]Format{
		"json": FormatJSON,
		"JSON": FormatJSON,
		"text": FormatText,
		"":     FormatText,
		"yaml": FormatText,
	}
	for input, want := range cases {
		if got := ParseFormat(input); got != want {
			t.Errorf("ParseFormat(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestInit_JSONFormatEmitsRecord(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{Level: slog.LevelDebug, Format: FormatJSON, Output: &buf})
	Info("hello", slog.String("k", "v"))

	out := buf.String()
	if !strings.Contains(out, `"msg":"hello"`) {
		t.Fatalf("expected JSON msg, got %q", out)
	}
	if !strings.Contains(out, `"k":"v"`) {
		t.Fatalf("expected JSON attr, got %q", out)
	}
}

func TestInit_TextFormatEmitsRecord(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{Level: slog.LevelDebug, Format: FormatText, Output: &buf})
	Warn("wat", slog.Int("n", 1))

	out := buf.String()
	if !strings.Contains(out, "msg=wat") {
		t.Fatalf("expected text msg, got %q", out)
	}
	if !strings.Contains(out, "n=1") {
		t.Fatalf("expected text attr, got %q", out)
	}
}

func TestInit_LevelFiltersOutLowerSeverity(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{Level: slog.LevelWarn, Format: FormatText, Output: &buf})
	Debug("not shown")
	Info("also not shown")
	Warn("shown")

	out := buf.String()
	if strings.Contains(out, "not shown") {
		t.Fatalf("debug/info should be filtered, got %q", out)
	}
	if !strings.Contains(out, "shown") {
		t.Fatalf("warn should be emitted, got %q", out)
	}
}

func TestDefault_ReturnsCurrent(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{Level: slog.LevelInfo, Format: FormatText, Output: &buf})
	if Default() == nil {
		t.Fatal("Default() returned nil")
	}
}

func TestWith_AddsAttrs(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{Level: slog.LevelInfo, Format: FormatJSON, Output: &buf})
	With(slog.String("component", "auth")).Info("ping")

	out := buf.String()
	if !strings.Contains(out, `"component":"auth"`) {
		t.Fatalf("expected component attr, got %q", out)
	}
}

func TestStdlibBridge_RedirectsStandardLog(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{Level: slog.LevelInfo, Format: FormatJSON, Output: &buf})
	log.Println("legacy log line")

	out := buf.String()
	if !strings.Contains(out, `"msg":"legacy log line"`) {
		t.Fatalf("expected bridged log line, got %q", out)
	}
}

func TestStdlibBridge_EmptyWriteIsNoop(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	b := stdlibBridge{logger: logger}

	n, err := b.Write([]byte("\n"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != 1 {
		t.Fatalf("Write returned %d, want 1", n)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected empty buffer, got %q", buf.String())
	}
}
