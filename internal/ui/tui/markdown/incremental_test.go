package markdown

import (
	"strings"
	"testing"
)

func TestRenderIncrementalCachesStableFencePart(t *testing.T) {
	content := "intro\n\n```go\nfmt.Println(\"hi\")\n```\n\nmore text"
	var cache SegmentCache

	first, err := RenderIncremental(content, 40, &cache)
	if err != nil {
		t.Fatal(err)
	}
	if cache.SegmentCacheLen() == 0 {
		t.Fatal("expected cached stable fence segment")
	}
	if !cache.SegmentCacheKeyPrefix("f:") {
		t.Fatal("expected fenced segment cache key")
	}

	growing := content + " and growing"
	second, err := RenderIncremental(growing, 40, &cache)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(second, "growing") {
		t.Fatalf("missing tail growth:\n%s", second)
	}
	if first == "" || second == "" {
		t.Fatal("expected non-empty renders")
	}
}

func TestRenderIncrementalCachesStableParagraphs(t *testing.T) {
	content := "para one\n\npara two\n\npara three"
	var cache SegmentCache

	_, err := RenderIncremental(content, 40, &cache)
	if err != nil {
		t.Fatal(err)
	}
	if StableFencePartCount(content) != 0 {
		t.Fatalf("unexpected fence parts")
	}
	if ParagraphStableCount(content) != 2 {
		t.Fatalf("stable paragraphs = %d, want 2", ParagraphStableCount(content))
	}
	if cache.SegmentCacheLen() < 2 {
		t.Fatalf("cache len = %d, want >= 2", cache.SegmentCacheLen())
	}
}

func TestRenderIncrementalUnclosedFenceRendersTail(t *testing.T) {
	content := "before\n\n```go\npartial"
	var cache SegmentCache

	out, err := RenderIncremental(content, 40, &cache)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "partial") {
		t.Fatalf("expected unclosed fence tail:\n%s", out)
	}
}

func TestHasUnclosedFence_proseWithoutFenceMarker(t *testing.T) {
	if ParagraphStableCount("para one\n\npara two\n\npara three") != 2 {
		t.Fatalf("stable paragraphs = %d, want 2", ParagraphStableCount("para one\n\npara two\n\npara three"))
	}
	if ParagraphStableCount("Use ``` syntax in prose") != 0 {
		t.Fatal("single paragraph with inline triple-backtick should not be treated as unclosed fence")
	}
	if ParagraphStableCount("para one\n\nUse ``` syntax in prose") != 1 {
		t.Fatalf("stable paragraphs = %d, want 1 (inline backtick must not disable cache)", ParagraphStableCount("para one\n\nUse ``` syntax in prose"))
	}
	if ParagraphStableCount("before\n\n```go\npartial") != 0 {
		t.Fatal("expected unclosed fence tail to disable paragraph cache")
	}
}

func TestRenderIncrementalNilCacheMatchesFullRender(t *testing.T) {
	content := "## Title\n\nHello **world**"
	full, err := Render(content, 40)
	if err != nil {
		t.Fatal(err)
	}
	inc, err := RenderIncremental(content, 40, nil)
	if err != nil {
		t.Fatal(err)
	}
	if full != inc {
		t.Fatalf("nil cache mismatch:\nfull=%q\ninc=%q", full, inc)
	}
}
