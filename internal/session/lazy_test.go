package session_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hejunqiu/ds-code/internal/session"
)

func TestLazyStore_defersInsertUntilMessage(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "sessions.db")
	inner, err := session.OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = inner.Close() })

	store := session.NewLazyStore(inner)
	ctx := context.Background()

	sess, err := store.CreateSession("deepseek-v4-pro", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}

	list, err := inner.ListSessions(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no persisted sessions, got %d", len(list))
	}

	if _, err := store.Get(ctx, sess.ID); err != nil {
		t.Fatalf("Get pending: %v", err)
	}

	if err := store.UpdateSession(ctx, sess.ID, func(s *session.Session) error {
		s.GitSnapshot = "branch main"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	list, err = inner.ListSessions(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("git snapshot update should not persist session, got %d rows", len(list))
	}

	if err := store.AppendMessage(ctx, session.Message{
		SessionID: sess.ID,
		Role:      "user",
		Content:   "hello",
	}); err != nil {
		t.Fatal(err)
	}

	list, err = inner.ListSessions(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("sessions after message = %d, want 1", len(list))
	}
	if list[0].ID != sess.ID {
		t.Fatalf("session id = %q, want %q", list[0].ID, sess.ID)
	}

	got, err := inner.Get(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.GitSnapshot != "branch main" {
		t.Fatalf("git_snapshot = %q, want branch main", got.GitSnapshot)
	}
}

func TestLazyStore_DropPending(t *testing.T) {
	t.Parallel()
	inner := session.NewMemoryStore()
	store := session.NewLazyStore(inner)
	ctx := context.Background()

	sess, err := store.CreateSession("m", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	session.DropPending(store, sess.ID)

	if _, err := store.Get(ctx, sess.ID); err == nil {
		t.Fatal("expected not found after drop")
	}
}
