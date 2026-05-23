package versioninfo

import (
	"runtime"
	"strings"
	"testing"
)

func TestPlatformForPrompt_includesOS(t *testing.T) {
	lines := PlatformForPrompt()
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %v", len(lines), lines)
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, runtime.GOOS) {
		t.Fatalf("expected GOOS %q in %q", runtime.GOOS, joined)
	}
	if !strings.HasPrefix(lines[0], "操作系统：") {
		t.Fatalf("first line: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "内核/架构：") {
		t.Fatalf("second line: %q", lines[1])
	}
}
