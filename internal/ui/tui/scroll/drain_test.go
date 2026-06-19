package scroll_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/scroll"
)

func TestDrainProportional_largeBurst(t *testing.T) {
	step := scroll.DrainStep(scroll.ProfileNative, 40, 20)
	if step < scroll.ScrollMinPerFrame {
		t.Fatalf("step = %d, want >= %d", step, scroll.ScrollMinPerFrame)
	}
	if step >= 20 {
		t.Fatalf("step = %d, should be capped below viewport", step)
	}
}

func TestDrainAdaptive_smallPending(t *testing.T) {
	step := scroll.DrainStep(scroll.ProfileIntegrated, 3, 10)
	if step != 3 {
		t.Fatalf("step = %d, want instant drain 3", step)
	}
}

func TestDrainAdaptive_snap(t *testing.T) {
	step := scroll.DrainStep(scroll.ProfileIntegrated, 50, 10)
	if step > scroll.ScrollMaxPending {
		t.Fatalf("step = %d, should not exceed snap cap", step)
	}
}

func TestClampPending(t *testing.T) {
	if got := scroll.ClampPending(100); got != scroll.PendingMax {
		t.Fatalf("got %d want %d", got, scroll.PendingMax)
	}
}

func TestSnapPending(t *testing.T) {
	if got := scroll.SnapPending(50); got != scroll.ScrollMaxPending {
		t.Fatalf("got %d want %d", got, scroll.ScrollMaxPending)
	}
}

func TestJumpYOffset_includesPending(t *testing.T) {
	got := scroll.JumpYOffset(10, 5, 20, 100)
	if got != 35 {
		t.Fatalf("got %d want 35", got)
	}
}
