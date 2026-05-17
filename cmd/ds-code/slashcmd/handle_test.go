package slashcmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
)

type mockHost struct{}

func (mockHost) SetRunMode(_ context.Context, _ *Env, _ string) error { return nil }

func testEnv(t *testing.T, store session.Store) (*Env, string) {
	t.Helper()
	sess, err := CreateSession(&config.Config{
		ProjectRoot: t.TempDir(),
		LLM:         config.LLMConfig{Model: "deepseek-v4-pro", ReasoningEffort: "high", Thinking: config.ThinkingConfig{Type: "enabled"}},
		Permission:  config.PermissionConfig{Mode: "readonly"},
		RunMode:     "agent",
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	id := sess.ID
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot: dir,
		LLM:         config.LLMConfig{Model: "deepseek-v4-pro", ReasoningEffort: "high", Thinking: config.ThinkingConfig{Type: "enabled"}},
		Permission:  config.PermissionConfig{Mode: "readonly"},
		RunMode:     "agent",
	}
	return &Env{
		Ctx:       context.Background(),
		Out:       &bytes.Buffer{},
		Cfg:       cfg,
		Store:     store,
		SessionID: &id,
		Runner: &agent.Runner{
			Perm:     permission.NewEngine(cfg.Permission.Mode, dir, false),
			Sessions: store,
		},
	}, id
}

func TestHandle_notSlash(t *testing.T) {
	env, _ := testEnv(t, session.NewMemoryStore())
	handled, err := Handle(env, mockHost{}, "hello")
	if err != nil || handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
}

func TestHandle_nilEnv(t *testing.T) {
	handled, err := Handle(nil, mockHost{}, "/help")
	if !handled || err == nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
}

func TestHandle_nilSessionID(t *testing.T) {
	store := session.NewMemoryStore()
	var buf bytes.Buffer
	env := &Env{
		Ctx:       context.Background(),
		Out:       &buf,
		Cfg:       &config.Config{ProjectRoot: t.TempDir()},
		Store:     store,
		SessionID: nil,
	}
	handled, err := Handle(env, mockHost{}, "/mode")
	if !handled || err == nil || !strings.Contains(err.Error(), "session not set") {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
}

func TestHandle_help(t *testing.T) {
	env, _ := testEnv(t, session.NewMemoryStore())
	handled, err := Handle(env, mockHost{}, "/help")
	if err != nil || !handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	out := env.Out.(*bytes.Buffer).String()
	if !strings.Contains(out, "/help") && !strings.Contains(out, "help") {
		t.Fatalf("help output missing commands: %q", out)
	}
}

func TestHandle_unknown(t *testing.T) {
	env, _ := testEnv(t, session.NewMemoryStore())
	handled, err := Handle(env, mockHost{}, "/not-a-real-cmd")
	if err != nil || !handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if !strings.Contains(env.Out.(*bytes.Buffer).String(), "Unknown command") {
		t.Fatal("expected unknown command message")
	}
}

func TestHandle_modeShow(t *testing.T) {
	env, _ := testEnv(t, session.NewMemoryStore())
	handled, err := Handle(env, mockHost{}, "/mode")
	if err != nil || !handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if !strings.Contains(env.Out.(*bytes.Buffer).String(), "model:") {
		t.Fatalf("output = %q", env.Out.(*bytes.Buffer).String())
	}
}
