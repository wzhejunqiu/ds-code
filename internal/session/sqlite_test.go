package session_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hejunqiu/ds-code/internal/session"
)

func TestSQLiteStore_roundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	store, err := session.OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sess, err := store.CreateSession("deepseek-v4-pro", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.AppendMessage(ctx, session.Message{
		SessionID: sess.ID,
		Role:      "user",
		Content:   "hello world from test",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title == "" {
		t.Fatal("expected title from first user message")
	}

	list, err := store.ListSessions(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("sessions = %d", len(list))
	}
}
