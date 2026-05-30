package app

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/runmode"
	"github.com/wzhejunqiu/ds-code/internal/session"
)

func TestApp_TrySlashLine_nilSession(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot: dir,
		LLM:         config.LLMConfig{Model: "m", ReasoningEffort: "high", Thinking: config.ThinkingConfig{Type: "enabled"}},
		Permission:  config.PermissionConfig{Mode: "readonly"},
		RunMode:     runmode.Agent,
	}
	a := New(cfg)
	store := session.NewMemoryStore()
	var buf bytes.Buffer
	handled, err := a.TrySlashLine(context.Background(), &buf, nil, store, nil, nil, "/mode")
	if !handled || err == nil || !strings.Contains(err.Error(), "session not set") {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
}
