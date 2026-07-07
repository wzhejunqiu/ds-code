package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
	"gopkg.in/yaml.v3"
)

func TestSavePermissionMode_user(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// Ensure ~/.ds-code/config exists path
	if err := config.SavePermissionMode("", false, permissionmode.Readonly); err != nil {
		t.Fatal(err)
	}
	path, err := config.UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	perm := doc["permission"].(map[string]any)
	if perm["mode"] != "readonly" {
		t.Fatalf("mode = %v", perm["mode"])
	}
}

func TestSavePermissionMode_project(t *testing.T) {
	root := t.TempDir()
	if err := config.SavePermissionMode(root, true, permissionmode.Ask); err != nil {
		t.Fatal(err)
	}
	path := config.ProjectConfigPath(root)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(path) {
		t.Fatal("expected abs path")
	}
	if len(b) == 0 {
		t.Fatal("empty config")
	}
}
