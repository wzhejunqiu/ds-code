package app

import (
	"context"
	"testing"

	"github.com/wzhejunqiu/ds-code/cmd/ds-code/slashcmd"
	"github.com/wzhejunqiu/ds-code/internal/config"
)

func TestApp_openStore(t *testing.T) {
	dir := t.TempDir()
	a := New(&config.Config{ProjectRoot: dir, LLM: config.LLMConfig{Model: "test"}})
	store, err := a.openStore()
	if err != nil {
		t.Fatal(err)
	}
	if store == nil {
		t.Fatal("expected store")
	}
	if a.store == nil {
		t.Fatal("expected cached lazy store")
	}
	store2, err := a.openStore()
	if err != nil || store2 != a.store {
		t.Fatalf("second open: store=%p cached=%p err=%v", store2, a.store, err)
	}
}

func TestApp_createSession(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot: dir,
		LLM: config.LLMConfig{
			Model:           "deepseek-v4-pro",
			ReasoningEffort: "max",
			Thinking:        config.ThinkingConfig{Type: "enabled"},
		},
		Permission: config.PermissionConfig{Mode: "readonly"},
		RunMode:    "agent",
	}
	a := New(cfg)
	store, err := a.openStore()
	if err != nil {
		t.Fatal(err)
	}
	sess, err := slashcmd.CreateSession(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(context.Background(), sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != cfg.LLM.Model || got.PermissionMode != "readonly" {
		t.Fatalf("session = %+v", got)
	}
}
