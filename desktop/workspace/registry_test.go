package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	desktopworkspace "github.com/wzhejunqiu/ds-code/desktop/workspace"
)

func TestRegistrySaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspaces.json")
	r, err := desktopworkspace.LoadRegistryAt(path)
	if err != nil {
		t.Fatal(err)
	}
	r.Upsert(desktopworkspace.Entry{ID: "abc", Root: "/tmp/proj", Name: "proj"})
	r.SetActive("abc")
	r.SetWindowLayout(desktopworkspace.WindowLayout{LeftWidth: 260, RightCollapsed: true})
	if err := r.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := desktopworkspace.LoadRegistryAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Active() != "abc" {
		t.Fatalf("active = %q", loaded.Active())
	}
	w := loaded.WindowLayout()
	if w.LeftWidth != 260 || !w.RightCollapsed {
		t.Fatalf("window layout = %+v", w)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestRegistryDedupeByRoot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspaces.json")
	r, err := desktopworkspace.LoadRegistryAt(path)
	if err != nil {
		t.Fatal(err)
	}
	r.Upsert(desktopworkspace.Entry{ID: "id1", Root: "/a/proj", Name: "proj"})
	r.Upsert(desktopworkspace.Entry{ID: "id2", Root: "/a/proj", Name: "renamed"})
	if len(r.Data().Workspaces) != 1 {
		t.Fatalf("workspaces = %d, want 1", len(r.Data().Workspaces))
	}
	if r.Data().Workspaces[0].Name != "renamed" {
		t.Fatalf("name = %q", r.Data().Workspaces[0].Name)
	}
}

func TestRegistryRemoveSwitchesActive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspaces.json")
	r, err := desktopworkspace.LoadRegistryAt(path)
	if err != nil {
		t.Fatal(err)
	}
	r.Upsert(desktopworkspace.Entry{ID: "a", Root: "/a", Name: "a"})
	r.Upsert(desktopworkspace.Entry{ID: "b", Root: "/b", Name: "b"})
	r.SetActive("a")
	if !r.Remove("a") {
		t.Fatal("remove failed")
	}
	if r.Active() != "b" {
		t.Fatalf("active = %q, want b", r.Active())
	}
}

func TestManagerAddAndList(t *testing.T) {
	root := t.TempDir()
	regPath := filepath.Join(t.TempDir(), "reg.json")
	reg, err := desktopworkspace.LoadRegistryAt(regPath)
	if err != nil {
		t.Fatal(err)
	}
	m, err := desktopworkspace.NewManagerWithRegistry(reg, nil)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := m.Add(root)
	if err != nil {
		t.Fatal(err)
	}
	if sum.ID == "" || !sum.Active {
		t.Fatalf("summary = %+v", sum)
	}
	list := m.List()
	if len(list) != 1 {
		t.Fatalf("list = %d", len(list))
	}
}
