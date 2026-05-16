package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSkill_projectDir(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".ds-code", "skills", "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Demo skill\nDo X."), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadSkill(dir, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if got == "" || !contains(got, "Demo skill") {
		t.Fatalf("skill content = %q", got)
	}
}

func TestListSkillNames(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".ds-code", "skills", "alpha")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}

	names, err := ListSkillNames(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "alpha" {
		t.Fatalf("names = %v", names)
	}
}

func TestLoadSkill_missing(t *testing.T) {
	_, err := LoadSkill(t.TempDir(), "nope")
	if err == nil {
		t.Fatal("expected error")
	}
}
