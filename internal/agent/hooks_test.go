package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHookManager_runsHook(t *testing.T) {
	dir := t.TempDir()
	dsCode := filepath.Join(dir, ".ds-code")
	if err := os.MkdirAll(dsCode, 0o700); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "hook.sh")
	if err := os.WriteFile(script, []byte(`#!/bin/sh
printf '%s' "$HOOK_INPUT"
`), 0o755); err != nil {
		t.Fatal(err)
	}
	hooksJSON := `[{"event":"PreToolUse","command":"` + script + `"}]`
	if err := os.WriteFile(filepath.Join(dsCode, "hooks.json"), []byte(hooksJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	hm := LoadHooks(dir)
	results := hm.Run(context.Background(), HookPreToolUse, `{"tool":"shell"}`)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Error != nil {
		t.Fatal(results[0].Error)
	}
	if results[0].Output != `{"tool":"shell"}` {
		t.Fatalf("unexpected output %q", results[0].Output)
	}
}

func TestHookManager_runsStopHook(t *testing.T) {
	dir := t.TempDir()
	dsCode := filepath.Join(dir, ".ds-code")
	if err := os.MkdirAll(dsCode, 0o700); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "stop.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	hooksJSON := `[{"event":"Stop","command":"` + script + `"}]`
	if err := os.WriteFile(filepath.Join(dsCode, "hooks.json"), []byte(hooksJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	hm := LoadHooks(dir)
	results := hm.Run(context.Background(), HookStop, `{"session_id":"s1"}`)
	if len(results) != 1 || results[0].Error != nil {
		t.Fatalf("Stop hook failed: %+v", results)
	}
}
