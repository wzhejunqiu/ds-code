package checkpoint_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hejunqiu/ds-code/internal/checkpoint"
)

func TestStoreCreateAndRewind(t *testing.T) {
	root := t.TempDir()
	store, err := checkpoint.OpenStore(root)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "foo.txt")
	if err := os.WriteFile(target, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	rec, err := store.Create(ctx, "sess-1", "write_file", []checkpoint.FileState{
		{RelPath: "foo.txt", Existed: true, Content: []byte("before")},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("after"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkpoint.ApplyRewind(root, rec); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "before" {
		t.Fatalf("got %q want before", data)
	}

	list, err := store.List(ctx, "sess-1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v err=%v", list, err)
	}
}

func TestCapturePaths_newFile(t *testing.T) {
	root := t.TempDir()
	resolve := func(rel string) (string, error) {
		return filepath.Join(root, rel), nil
	}
	states, err := checkpoint.CapturePaths(root, resolve, []string{"new.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 1 || states[0].Existed {
		t.Fatalf("states = %+v", states)
	}
}
