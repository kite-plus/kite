package service

import "testing"

func TestNormalizeDangerousExtensionRules(t *testing.T) {
	normalized, err := NormalizeDangerousExtensionRules(`[{"ext":" .EXE ","action":"rename"},{"ext":".Svg","action":"block"}]`)
	if err != nil {
		t.Fatalf("NormalizeDangerousExtensionRules: %v", err)
	}

	if normalized != `[{"ext":".exe","action":"rename"},{"ext":".svg","action":"block"}]` {
		t.Fatalf("unexpected normalized rules: %q", normalized)
	}
}

func TestNormalizeDangerousExtensionRulesRejectsInvalidInput(t *testing.T) {
	cases := []string{
		`[{"ext":"exe","action":"block"}]`,
		`[{"ext":".exe","action":"noop"}]`,
		`[{"ext":".exe","action":"block"},{"ext":".EXE","action":"rename"}]`,
	}

	for _, raw := range cases {
		if _, err := NormalizeDangerousExtensionRules(raw); err == nil {
			t.Fatalf("expected invalid rules for %q", raw)
		}
	}
}

func TestNormalizeDangerousRenameSuffix(t *testing.T) {
	normalized, err := NormalizeDangerousRenameSuffix(" .SAFE_File ")
	if err != nil {
		t.Fatalf("NormalizeDangerousRenameSuffix: %v", err)
	}

	if normalized != "safe_file" {
		t.Fatalf("unexpected normalized suffix: %q", normalized)
	}
}

func TestNormalizeDangerousRenameSuffixRejectsInvalidInput(t *testing.T) {
	if _, err := NormalizeDangerousRenameSuffix("bad suffix"); err == nil {
		t.Fatal("expected invalid suffix error")
	}
}

func TestDecideDangerousExtensionRename(t *testing.T) {
	decision, matched, err := DecideDangerousExtension("setup.EXE", []DangerousExtensionRule{{
		Ext:    ".exe",
		Action: DangerousExtensionActionRename,
	}}, "blocked")
	if err != nil {
		t.Fatalf("DecideDangerousExtension: %v", err)
	}
	if !matched {
		t.Fatal("expected dangerous extension to match")
	}
	if decision.SafeName != "setup.EXE.blocked" {
		t.Fatalf("unexpected safe name: %q", decision.SafeName)
	}
	if decision.SafeExt != "blocked" {
		t.Fatalf("unexpected safe ext: %q", decision.SafeExt)
	}
}
