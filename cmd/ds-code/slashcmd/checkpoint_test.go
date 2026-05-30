package slashcmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/runmode"
	"github.com/wzhejunqiu/ds-code/internal/session"
)

func TestParseRewindArgs(t *testing.T) {
	id, yes, err := parseRewindArgs("3 --yes")
	if err != nil || id != 3 || !yes {
		t.Fatalf("id=%d yes=%v err=%v", id, yes, err)
	}
	_, yes, err = parseRewindArgs("3")
	if err != nil || yes {
		t.Fatalf("unexpected yes=%v err=%v", yes, err)
	}
	_, _, err = parseRewindArgs("nope")
	if err == nil {
		t.Fatal("expected error for invalid id")
	}
}

func TestCheckpointRewind_requiresYes(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := CreateSession(&config.Config{
		ProjectRoot: t.TempDir(),
		LLM:         config.LLMConfig{Model: "m", ReasoningEffort: "high", Thinking: config.ThinkingConfig{Type: "enabled"}},
		Permission:  config.PermissionConfig{Mode: "readonly"},
		RunMode:     runmode.Agent,
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	id := sess.ID
	env := &Env{
		Ctx:       context.Background(),
		Out:       &buf,
		Cfg:       &config.Config{ProjectRoot: t.TempDir()},
		Store:     store,
		SessionID: &id,
		Runner:    &agent.Runner{Sessions: store},
	}
	if err := checkpointRewind(env, "1"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "--yes") {
		t.Fatalf("expected confirmation prompt, got %q", out)
	}
}

func TestCheckpointRewind_nilRunner(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := CreateSession(&config.Config{
		ProjectRoot: t.TempDir(),
		LLM:         config.LLMConfig{Model: "m", ReasoningEffort: "high", Thinking: config.ThinkingConfig{Type: "enabled"}},
		Permission:  config.PermissionConfig{Mode: "readonly"},
		RunMode:     runmode.Agent,
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	id := sess.ID
	env := &Env{
		Ctx:       context.Background(),
		Out:       &buf,
		Cfg:       &config.Config{ProjectRoot: t.TempDir()},
		Store:     store,
		SessionID: &id,
	}
	err = checkpointRewind(env, "1 --yes")
	if err == nil || !strings.Contains(err.Error(), "nil runner") {
		t.Fatalf("err = %v", err)
	}
}
