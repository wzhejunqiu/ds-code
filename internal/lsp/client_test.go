package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/lsp/transport"
)

func TestPathToURI(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	uri := pathToURI(file)
	if !strings.HasPrefix(uri, "file://") {
		t.Fatalf("uri = %q", uri)
	}
	if !strings.Contains(uri, "main.go") {
		t.Fatalf("uri = %q", uri)
	}
}

func TestLanguageID(t *testing.T) {
	cases := map[string]string{
		"main.go":   "go",
		"app.ts":    "typescript",
		"app.tsx":   "typescript",
		"app.js":    "javascript",
		"app.py":    "python",
		"Main.java": "java",
		"lib.rs":    "rust",
		"readme":    "plaintext",
	}
	for path, want := range cases {
		if got := languageID(path); got != want {
			t.Fatalf("languageID(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestFilterDiags_severityAndMax(t *testing.T) {
	in := []Diagnostic{
		{Severity: "error", Message: "e1"},
		{Severity: "warning", Message: "w1"},
		{Severity: "info", Message: "i1"},
	}
	sev := map[string]bool{"error": true}
	out := filterDiags(in, sev, 0)
	if len(out) != 1 || out[0].Message != "e1" {
		t.Fatalf("filtered = %+v", out)
	}
	out = filterDiags(in, nil, 2)
	if len(out) != 2 {
		t.Fatalf("max 2: got %+v", out)
	}
}

func TestClient_readLoop_publishDiagnostics(t *testing.T) {
	c := NewClient(t.TempDir(), config.LSPConfig{}, ServerConfig{ID: "go"})
	pr, pw := io.Pipe()
	c.reader = bufio.NewReader(pr)

	done := make(chan struct{})
	go func() {
		c.readLoop()
		close(done)
	}()

	uri := pathToURI(filepath.Join(c.root, "main.go"))
	err := transport.WriteMessage(pw, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/publishDiagnostics",
		"params": map[string]any{
			"uri": uri,
			"diagnostics": []map[string]any{
				{
					"range": map[string]any{
						"start": map[string]any{"line": 0, "character": 0},
					},
					"severity": 2,
					"message":  "unused var",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.After(2 * time.Second)
	for {
		c.mu.Lock()
		n := len(c.diags[uri])
		c.mu.Unlock()
		if n > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for diagnostics")
		case <-time.After(10 * time.Millisecond):
		}
	}

	c.mu.Lock()
	d := c.diags[uri][0]
	c.mu.Unlock()
	if d.Severity != "warning" || d.Message != "unused var" {
		t.Fatalf("diag = %+v", d)
	}
	_ = pw.Close()
	<-done
}

func TestClient_initialize_mocked(t *testing.T) {
	c := NewClient(t.TempDir(), config.LSPConfig{DiagnosticsTimeout: 5 * time.Second}, ServerConfig{ID: "go"})
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	c.stdin = stdinW
	c.reader = bufio.NewReader(stdoutR)
	go c.readLoop()

	go func() {
		reader := bufio.NewReader(stdinR)
		for {
			msg, err := transport.ReadMessage(reader)
			if err != nil {
				return
			}
			var env struct {
				ID     *int   `json:"id"`
				Method string `json:"method"`
			}
			_ = json.Unmarshal(msg, &env)
			if env.Method == "initialize" && env.ID != nil {
				_ = transport.WriteMessage(stdoutW, map[string]any{
					"jsonrpc": "2.0",
					"id":      *env.ID,
					"result":  map[string]any{"capabilities": map[string]any{}},
				})
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.initialize(ctx); err != nil {
		t.Fatal(err)
	}
	_ = stdinW.Close()
	_ = stdoutW.Close()
}
