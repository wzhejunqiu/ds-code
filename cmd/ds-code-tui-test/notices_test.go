//go:build tuitest

package main

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/header"
)

func TestHarnessStartupNotices_scrollable(t *testing.T) {
	notices := harnessStartupNotices()
	if len(notices) < 3 {
		t.Fatalf("expected multiple demo notices, got %d", len(notices))
	}
	zoneW := header.ZoneWidth(120, false)
	if header.MaxScrollOffset(notices, zoneW) <= 0 {
		t.Fatal("harness demo notices should exceed visible window and auto-scroll")
	}
	text := header.Render(120, "tui-test", nil, nil, 0, notices, 0)
	if !strings.Contains(text, "MCP 跳过") {
		t.Fatalf("missing MCP summary in header: %q", text)
	}
}
