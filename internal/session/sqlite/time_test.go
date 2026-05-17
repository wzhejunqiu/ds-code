package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStore_badTimestamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sess, err := store.CreateSession("m", "high", "enabled", "readonly", "agent")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec(`UPDATE sessions SET created_at='not-a-time' WHERE id=?`, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Get(context.Background(), sess.ID)
	if err == nil {
		t.Fatal("expected parse error for corrupted created_at")
	}
}

func TestParseRFC3339_empty(t *testing.T) {
	got, err := parseRFC3339("")
	if err != nil || !got.IsZero() {
		t.Fatalf("got=%v err=%v", got, err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	got, err = parseRFC3339(now)
	if err != nil || got.IsZero() {
		t.Fatalf("got=%v err=%v", got, err)
	}
}
