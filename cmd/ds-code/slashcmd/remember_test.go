package slashcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
)

func TestRemember_activeAgentType(t *testing.T) {
	dir := testutil.IsolatedHome(t)

	var out bytes.Buffer
	env := &Env{
		Out:             &out,
		Cfg:             &config.Config{ProjectRoot: dir},
		ActiveAgentType: "Explore",
	}
	if err := Remember(env, "user prefers tabs"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".ds-code", "agent-memory", "Explore", "user.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "prefers tabs") {
		t.Fatalf("unexpected content %q", string(data))
	}
}

func TestRemember_appendsSameSlot(t *testing.T) {
	dir := testutil.IsolatedHome(t)

	var out bytes.Buffer
	env := &Env{
		Out: &out,
		Cfg: &config.Config{ProjectRoot: dir},
	}
	if err := Remember(env, "user first"); err != nil {
		t.Fatal(err)
	}
	if err := Remember(env, "user second"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".ds-code", "agent-memory", "general-purpose", "user.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "first") || !strings.Contains(body, "second") {
		t.Fatalf("expected appended content, got %q", body)
	}
}
