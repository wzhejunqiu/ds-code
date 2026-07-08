package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
)

func TestSaveLLMSettings_user(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := SaveLLMSettings("", false, "deepseek-v4-flash", "high"); err != nil {
		t.Fatal(err)
	}
	path, err := UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !containsAll(text, "deepseek-v4-flash", "high") {
		t.Fatalf("yaml = %q", text)
	}
}

func TestSaveLLMSettings_project(t *testing.T) {
	root := t.TempDir()
	if err := SaveLLMSettings(root, true, "deepseek-v4-pro", "max"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, ".ds-code", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !containsAll(text, "deepseek-v4-pro", "max") {
		t.Fatalf("yaml = %q", text)
	}
}

func TestSaveLLMSettings_invalidModel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if err := SaveLLMSettings("", false, "invalid-model", ""); err == nil {
		t.Fatal("expected error for invalid model")
	}
}

func TestSaveTracingSettings_user(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := SaveTracingSettings("", false, true, "log", ""); err != nil {
		t.Fatal(err)
	}
	path, err := UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !containsAll(text, "enabled: true", "exporter: log") {
		t.Fatalf("yaml = %q", text)
	}
}

func TestSaveTracingSettings_otlpRequiresEndpoint(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if err := SaveTracingSettings("", false, true, "otlp", ""); err == nil {
		t.Fatal("expected error when otlp endpoint missing")
	}
}

func TestSaveGeneralSettings_permissionOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if err := SaveGeneralSettings("", false, permissionmode.Auto, "", "", nil, "", ""); err != nil {
		t.Fatal(err)
	}
	path, err := UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(b), "mode: auto") {
		t.Fatalf("yaml = %q", string(b))
	}
}

func containsAll(text string, parts ...string) bool {
	for _, p := range parts {
		if !contains(text, p) {
			return false
		}
	}
	return true
}

func contains(text, sub string) bool {
	return len(text) >= len(sub) && (text == sub || len(sub) == 0 || indexOf(text, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
