package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/session/sqlite"
)

func TestStore_roundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	store, err := sqlite.Open(path)
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
		Role:      role.User,
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

func TestStore_messageDurations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sess, err := store.CreateSession("m", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.AppendMessage(ctx, session.Message{
		SessionID:           sess.ID,
		Role:                role.Assistant,
		Content:             "hi",
		ReasoningContent:    "think",
		ReasoningDurationMS: 1500,
		TurnDurationMS:      4200,
	}); err != nil {
		t.Fatal(err)
	}
	msgs, err := store.ListMessages(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages = %d", len(msgs))
	}
	if msgs[0].ReasoningDurationMS != 1500 || msgs[0].TurnDurationMS != 4200 {
		t.Fatalf("durations: reasoning=%d turn=%d", msgs[0].ReasoningDurationMS, msgs[0].TurnDurationMS)
	}
}
