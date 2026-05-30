package chat

import (
	"strings"
	"testing"
	"time"
)

func TestRenderCacheHitOnStableBlock(t *testing.T) {
	blocks := []Block{
		{Role: RoleUser, Content: "hello"},
		{Role: RoleAssistant, Content: "world"},
	}
	now := time.Now()
	var cache RenderCache

	first := RenderCached(blocks, 60, now, false, &cache, nil)
	second := RenderCached(blocks, 60, now, false, &cache, nil)
	if first != second {
		t.Fatalf("expected cache hit to produce identical output")
	}
	if len(cache.entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(cache.entries))
	}
	if !strings.Contains(first, "hello") || !strings.Contains(first, "world") {
		t.Fatalf("unexpected output:\n%s", first)
	}
}

func TestRenderCacheInvalidatesOnContentChange(t *testing.T) {
	blocks := []Block{{Role: RoleAssistant, Content: "a"}}
	now := time.Now()
	var cache RenderCache

	first := RenderCached(blocks, 60, now, false, &cache, nil)
	blocks[0].Content = "ab"
	second := RenderCached(blocks, 60, now, false, &cache, nil)
	if first == second {
		t.Fatal("expected cache miss after content change")
	}
}

func TestRenderCacheLiveBlockUsesTimeBucket(t *testing.T) {
	t0 := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	started := t0.Add(-500 * time.Millisecond)
	blocks := []Block{{
		Role:               RoleAssistant,
		ReasoningStartedAt: started,
		Streaming:          true,
	}}
	blocks[0].Reasoning = "thinking"

	var cache RenderCache
	out0 := RenderCached(blocks, 60, t0, false, &cache, nil)
	key0 := cache.entries[0].key

	out1 := RenderCached(blocks, 60, t0.Add(50*time.Millisecond), false, &cache, nil)
	if cache.entries[0].key != key0 {
		t.Fatal("expected same 100ms bucket within 50ms")
	}
	if out0 != out1 {
		t.Fatal("expected same render within time bucket")
	}

	RenderCached(blocks, 60, t0.Add(150*time.Millisecond), false, &cache, nil)
	if cache.entries[0].key == key0 {
		t.Fatal("expected new bucket after 150ms")
	}
}

func TestRenderCacheResetOnWidthChange(t *testing.T) {
	blocks := []Block{{Role: RoleUser, Content: "x"}}
	now := time.Now()
	var cache RenderCache

	RenderCached(blocks, 60, now, false, &cache, nil)
	RenderCached(blocks, 80, now, false, &cache, nil)
	if cache.width != 80 {
		t.Fatalf("width = %d, want 80", cache.width)
	}
}

func TestRenderCacheResizeClearsStaleSlots(t *testing.T) {
	now := time.Now()
	var cache RenderCache

	blocks10 := make([]Block, 10)
	for i := range blocks10 {
		blocks10[i] = Block{Role: RoleUser, Content: strings.Repeat("a", i+1)}
	}
	RenderCached(blocks10, 60, now, false, &cache, nil)

	blocks5 := blocks10[:5]
	RenderCached(blocks5, 60, now, false, &cache, nil)

	blocks8 := make([]Block, 8)
	copy(blocks8, blocks5)
	blocks8[5] = Block{Role: RoleUser, Content: "unique-slot-5"}
	blocks8[6] = Block{Role: RoleUser, Content: "unique-slot-6"}
	blocks8[7] = Block{Role: RoleUser, Content: "unique-slot-7"}

	out := RenderCached(blocks8, 60, now, false, &cache, nil)
	for _, want := range []string{"unique-slot-5", "unique-slot-6", "unique-slot-7"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output after resize:\n%s", want, out)
		}
	}
	if len(cache.entries) != 8 {
		t.Fatalf("entries = %d, want 8", len(cache.entries))
	}
}
