package scroll_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/scroll"
)

func TestWheelSpeed_default(t *testing.T) {
	t.Setenv("DS_CODE_SCROLL_SPEED", "")
	if got := scroll.WheelSpeed(); got != 1 {
		t.Fatalf("speed = %v, want 1", got)
	}
}

func TestWheelSpeed_env(t *testing.T) {
	t.Setenv("DS_CODE_SCROLL_SPEED", "2")
	if got := scroll.WheelSpeed(); got != 2 {
		t.Fatalf("speed = %v, want 2", got)
	}
}

func TestWheelSpeed_invalid(t *testing.T) {
	t.Setenv("DS_CODE_SCROLL_SPEED", "not-a-number")
	if got := scroll.WheelSpeed(); got != 1 {
		t.Fatalf("speed = %v, want default 1", got)
	}
}
