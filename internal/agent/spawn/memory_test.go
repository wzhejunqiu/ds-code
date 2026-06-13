package spawn

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/testutil"
)

func TestLoadAgentMemory_truncatesAndPicksRecent(t *testing.T) {
	dir := testutil.IsolatedHome(t)

	agentDir := filepath.Join(dir, ".ds-code", "agent-memory", "Explore")
	if err := os.MkdirAll(agentDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "user.md"), []byte("user prefs"), 0o600); err != nil {
		t.Fatal(err)
	}
	body, err := LoadAgentMemory("Explore")
	if err != nil {
		t.Fatal(err)
	}
	if body == "" || !containsStr(body, "user prefs") {
		t.Fatalf("expected memory body, got %q", body)
	}
}

func TestSaveAgentMemory_appends(t *testing.T) {
	dir := testutil.IsolatedHome(t)

	if err := SaveAgentMemory("Explore", "user", "first note"); err != nil {
		t.Fatal(err)
	}
	if err := SaveAgentMemory("Explore", "user", "second note"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".ds-code", "agent-memory", "Explore", "user.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !containsStr(body, "first note") || !containsStr(body, "second note") {
		t.Fatalf("expected both notes appended, got %q", body)
	}
}

func containsStr(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexStr(s, sub) >= 0)
}

func indexStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
