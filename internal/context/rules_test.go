package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRules(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".ds-code", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "style.md"), []byte("Use Go idioms."), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRules(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" || !contains(got, "Use Go idioms") {
		t.Fatalf("unexpected rules: %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
