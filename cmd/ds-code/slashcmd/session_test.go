package slashcmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
)

func TestPermissions_showAndSet(t *testing.T) {
	store := session.NewMemoryStore()
	env, _ := testEnv(t, store)
	var buf bytes.Buffer
	env.Out = &buf

	if err := Permissions(env, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "permission.mode:") {
		t.Fatalf("show: %q", buf.String())
	}

	buf.Reset()
	if err := Permissions(env, "readonly"); err != nil {
		t.Fatal(err)
	}
	if env.Runner.Perm.Mode != "readonly" {
		t.Fatalf("mode = %q", env.Runner.Perm.Mode)
	}
}

func TestResume_listAndSwitch(t *testing.T) {
	store := session.NewMemoryStore()
	env, id := testEnv(t, store)
	var buf bytes.Buffer
	env.Out = &buf

	sess2, err := CreateSession(env.Cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	_ = store.AppendMessage(context.Background(), session.Message{
		SessionID: sess2.ID, Role: role.User, Content: "hello",
	})

	buf.Reset()
	if err := Resume(env, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), sess2.ID) {
		t.Fatalf("list: %q", buf.String())
	}

	buf.Reset()
	if err := Resume(env, sess2.ID); err != nil {
		t.Fatal(err)
	}
	if *env.SessionID != sess2.ID {
		t.Fatalf("session id = %q want %q", *env.SessionID, sess2.ID)
	}
	_ = id
}

func TestPermissions_autoRequiresYes(t *testing.T) {
	env, _ := testEnv(t, session.NewMemoryStore())
	var buf bytes.Buffer
	env.Out = &buf

	if err := Permissions(env, "auto"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "--yes") {
		t.Fatalf("expected confirmation hint: %q", buf.String())
	}
	if env.Runner.Perm.Mode == "auto" {
		t.Fatal("auto should not apply without --yes")
	}

	buf.Reset()
	if err := Permissions(env, "auto --yes"); err != nil {
		t.Fatal(err)
	}
	if env.Runner.Perm.Mode != "auto" {
		t.Fatalf("mode = %q", env.Runner.Perm.Mode)
	}
}

func TestMode_invalid(t *testing.T) {
	env, _ := testEnv(t, session.NewMemoryStore())
	err := Mode(env, "not-a-model")
	if err == nil || !strings.Contains(err.Error(), "invalid model") {
		t.Fatalf("err = %v", err)
	}
}
