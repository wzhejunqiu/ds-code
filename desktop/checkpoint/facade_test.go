package checkpoint_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	desktopcp "github.com/wzhejunqiu/ds-code/desktop/checkpoint"
	icp "github.com/wzhejunqiu/ds-code/internal/checkpoint"
)

func TestPreviewRewind_showsCurrentToCheckpoint(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "foo.txt")
	if err := os.WriteFile(target, []byte("current"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := icp.Record{
		ID: 1,
		Files: []icp.FileState{
			{RelPath: "foo.txt", Existed: true, Content: []byte("before")},
		},
	}
	diffs, err := desktopcp.PreviewRewind(root, rec)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("diffs = %d, want 1", len(diffs))
	}
	if diffs[0].Original != "current" || diffs[0].Modified != "before" {
		t.Fatalf("diff = %+v", diffs[0])
	}
}

func TestListAndNewerIDs(t *testing.T) {
	root := t.TempDir()
	store, err := icp.OpenAt(filepath.Join(root, "checkpoints"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := store.Create(ctx, "s1", "apply_patch", []icp.FileState{
		{RelPath: "a.go", Existed: true, Content: []byte("a")},
	}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, "s1", "apply_patch", []icp.FileState{
		{RelPath: "b.go", Existed: true, Content: []byte("b")},
	}, ""); err != nil {
		t.Fatal(err)
	}
	list, err := desktopcp.List(ctx, store, "s1")
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %v err=%v", list, err)
	}
	ids, err := desktopcp.NewerIDs(ctx, store, "s1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 2 {
		t.Fatalf("newer = %v", ids)
	}
}
