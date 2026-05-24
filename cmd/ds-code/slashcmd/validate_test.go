package slashcmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/session"
)

func TestRequireSessionEnv(t *testing.T) {
	if err := requireSessionEnv(nil); err == nil {
		t.Fatal("expected error for nil env")
	}
	env := &Env{Out: &bytes.Buffer{}, Cfg: &config.Config{}}
	if err := requireSessionEnv(env); err == nil || !strings.Contains(err.Error(), "session store") {
		t.Fatalf("err = %v", err)
	}
}

func TestHandle_clear(t *testing.T) {
	store := session.NewMemoryStore()
	env, oldID := testEnv(t, store)
	env.Cfg.ProjectRoot = t.TempDir()
	handled, err := Handle(env, mockHost{}, "/clear")
	if err != nil || !handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if *env.SessionID == oldID {
		t.Fatal("expected new session id after /clear")
	}
	if !strings.Contains(env.Out.(*bytes.Buffer).String(), "New session:") {
		t.Fatalf("output = %q", env.Out.(*bytes.Buffer).String())
	}
}

func TestHandle_git_noRepo(t *testing.T) {
	env, _ := testEnv(t, session.NewMemoryStore())
	env.Cfg.ProjectRoot = t.TempDir()
	env.CtxSvc = nil // caught by handle
	handled, err := Handle(env, mockHost{}, "/git")
	if !handled || err == nil || !strings.Contains(err.Error(), "context service") {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	_ = context.Background()
}
