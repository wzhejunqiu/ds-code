package scroll

import "testing"

func TestIsPinnedBottom(t *testing.T) {
	if IsPinnedBottom(ChatBottomSentinel - 1) {
		t.Fatal("value below sentinel should not be pinned")
	}
	if !IsPinnedBottom(ChatBottomSentinel) {
		t.Fatal("sentinel should be pinned")
	}
}

func TestEffectiveChatY(t *testing.T) {
	if got := EffectiveChatY(ChatBottomSentinel, 42); got != 42 {
		t.Fatalf("sentinel: got %d want 42", got)
	}
	if got := EffectiveChatY(-5, 10); got != 0 {
		t.Fatalf("negative: got %d want 0", got)
	}
	if got := EffectiveChatY(99, 10); got != 10 {
		t.Fatalf("above max: got %d want 10", got)
	}
	if got := EffectiveChatY(3, 10); got != 3 {
		t.Fatalf("in range: got %d want 3", got)
	}
}
