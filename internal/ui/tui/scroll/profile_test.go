package scroll_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/scroll"
)

func TestDetectProfile_integrated(t *testing.T) {
	t.Setenv("VSCODE_INJECTED", "1")
	t.Setenv("TERM_PROGRAM", "")
	if got := scroll.DetectProfile(); got != scroll.ProfileIntegrated {
		t.Fatalf("profile = %v, want integrated", got)
	}
}

func TestDetectProfile_cursor(t *testing.T) {
	t.Setenv("VSCODE_INJECTED", "")
	t.Setenv("TERM_PROGRAM", "cursor")
	if got := scroll.DetectProfile(); got != scroll.ProfileIntegrated {
		t.Fatalf("profile = %v, want integrated", got)
	}
}

func TestDetectProfile_native(t *testing.T) {
	t.Setenv("VSCODE_INJECTED", "")
	t.Setenv("TERM_PROGRAM", "Apple_Terminal")
	if got := scroll.DetectProfile(); got != scroll.ProfileNative {
		t.Fatalf("profile = %v, want native", got)
	}
}
