package glob_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
)

// Integration I/O tests (G8–G15): exact input JSON → output or error.

func TestGlobTool_IO_exactOutput(t *testing.T) {
	sameTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("G8_single_match", func(t *testing.T) {
		f := newGlobFixture(t)
		f.write(t, "only.go", "package only\n")
		out := f.exec(t, map[string]any{"pattern": "*.go"})
		if out != "Found 1 files\nonly.go" {
			t.Fatalf("got %q", out)
		}
	})

	t.Run("G9_two_files_same_mtime_sorted_by_path", func(t *testing.T) {
		f := newGlobFixture(t)
		f.write(t, "a.go", "package a\n")
		f.write(t, "pkg/b.go", "package b\n")
		f.chtimes(t, "a.go", sameTime)
		f.chtimes(t, "pkg/b.go", sameTime)
		out := f.exec(t, map[string]any{"pattern": "**/*.go"})
		want := "Found 2 files\na.go\npkg/b.go"
		if out != want {
			t.Fatalf("got %q want %q", out, want)
		}
	})

	t.Run("G10_no_match", func(t *testing.T) {
		f := newGlobFixture(t)
		f.write(t, "readme.txt", "hi\n")
		out := f.exec(t, map[string]any{"pattern": "*.go"})
		if out != "Found 0 files" {
			t.Fatalf("got %q", out)
		}
	})

	t.Run("G11_path_scoped_workspace_relative", func(t *testing.T) {
		f := newGlobFixture(t)
		f.write(t, "internal/pkg/a.go", "package pkg\n")
		out := f.exec(t, map[string]any{
			"pattern": "*.go",
			"path":    "internal/pkg",
		})
		if out != "Found 1 files\ninternal/pkg/a.go" {
			t.Fatalf("got %q", out)
		}
	})

	t.Run("G12_implicit_path_no_validation", func(t *testing.T) {
		f := newGlobFixture(t)
		f.write(t, "ok.go", "package ok\n")
		// No path key — should succeed even though we never created "missing/" dir.
		out := f.exec(t, map[string]any{"pattern": "*.go"})
		if !strings.HasPrefix(out, "Found 1 files") {
			t.Fatalf("got %q", out)
		}
	})
}

func TestGlobTool_IO_inputErrors(t *testing.T) {
	t.Run("G13_empty_pattern", func(t *testing.T) {
		f := newGlobFixture(t)
		raw, _ := json.Marshal(map[string]any{"pattern": ""})
		_, err := f.g.Execute(t.Context(), raw)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), builtin.ErrPatternRequired) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("G14_explicit_path_file", func(t *testing.T) {
		f := newGlobFixture(t)
		f.write(t, "single.go", "package x\n")
		raw, _ := json.Marshal(map[string]any{"pattern": "*.go", "path": "single.go"})
		_, err := f.g.Execute(t.Context(), raw)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "必须是目录") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("G15_explicit_path_missing", func(t *testing.T) {
		f := newGlobFixture(t)
		raw, _ := json.Marshal(map[string]any{"pattern": "*.go", "path": "missing/dir"})
		_, err := f.g.Execute(t.Context(), raw)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "目录不存在") {
			t.Fatalf("got %v", err)
		}
	})
}
