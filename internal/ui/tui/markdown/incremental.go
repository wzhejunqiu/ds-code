package markdown

import (
	"fmt"
	"strconv"
	"strings"
)

// SegmentCache caches stable markdown segments during streaming render.
type SegmentCache struct {
	width       int
	stableParts []segmentEntry
}

type segmentEntry struct {
	key      string
	rendered string
}

// Reset clears all cached segments.
func (c *SegmentCache) Reset() {
	if c == nil {
		return
	}
	c.stableParts = nil
	c.width = 0
}

// RenderIncremental renders content reusing cached stable fence/paragraph segments.
// When cache is nil, falls back to full Render.
func RenderIncremental(content string, width int, cache *SegmentCache) (string, error) {
	if cache == nil {
		return renderFull(content, width)
	}
	if width < 1 {
		width = 1
	}
	if cache.width != width {
		cache.Reset()
		cache.width = width
	}

	parts := splitByFences(content)
	if len(parts) == 0 {
		return "", nil
	}

	var b strings.Builder
	for i, part := range parts {
		isTail := i == len(parts)-1
		if !isTail {
			key := stablePartKey(width, part)
			if rendered, ok := cache.lookupStable(key); ok {
				b.WriteString(rendered)
				continue
			}
			rendered, err := renderPart(part, width)
			if err != nil {
				return "", err
			}
			cache.storeStable(key, rendered)
			b.WriteString(rendered)
			continue
		}

		rendered, err := renderTailPart(part, width, cache)
		if err != nil {
			return "", err
		}
		b.WriteString(rendered)
	}
	return normalizeOutput(b.String()), nil
}

func (c *SegmentCache) lookupStable(key string) (string, bool) {
	for _, e := range c.stableParts {
		if e.key == key {
			return e.rendered, true
		}
	}
	return "", false
}

func (c *SegmentCache) storeStable(key, rendered string) {
	for i := range c.stableParts {
		if c.stableParts[i].key == key {
			c.stableParts[i].rendered = rendered
			return
		}
	}
	c.stableParts = append(c.stableParts, segmentEntry{key: key, rendered: rendered})
}

func stablePartKey(width int, p part) string {
	if p.fenced {
		return fmt.Sprintf("f:%d:%s:%s", width, p.lang, p.code)
	}
	return fmt.Sprintf("p:%d:%s", width, p.text)
}

func renderPart(p part, width int) (string, error) {
	segment := p.text
	innerWidth := width
	if p.fenced {
		segment = fencedMarkdown(p.lang, p.code)
		innerWidth = codeBlockInnerWidth(width)
	}
	rendered, err := renderSegment(segment, innerWidth)
	if err != nil {
		return "", err
	}
	if p.fenced {
		rendered = boxRenderedCodeBlock(rendered)
	}
	return rendered, nil
}

func renderTailPart(p part, width int, cache *SegmentCache) (string, error) {
	if p.fenced || hasUnclosedFence(p.text) {
		return renderPart(p, width)
	}
	if strings.TrimSpace(p.text) == "" {
		return "", nil
	}

	paras := splitParagraphs(p.text)
	if len(paras) <= 1 {
		return renderPart(p, width)
	}

	var b strings.Builder
	for i, para := range paras {
		isTail := i == len(paras)-1
		if !isTail {
			key := fmt.Sprintf("para:%d:%s", width, para)
			if rendered, ok := cache.lookupStable(key); ok {
				b.WriteString(rendered)
				continue
			}
			rendered, err := renderPart(part{text: para}, width)
			if err != nil {
				return "", err
			}
			cache.storeStable(key, rendered)
			b.WriteString(rendered)
			continue
		}
		rendered, err := renderPart(part{text: para}, width)
		if err != nil {
			return "", err
		}
		b.WriteString(rendered)
	}
	return b.String(), nil
}

func hasUnclosedFence(text string) bool {
	inFence := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !inFence {
			if isFenceOpenLine(trimmed) {
				inFence = true
			}
			continue
		}
		if trimmed == "```" {
			inFence = false
		}
	}
	return inFence
}

func splitParagraphs(text string) []string {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil
	}
	raw := strings.Split(text, "\n\n")
	out := make([]string, 0, len(raw))
	for i, p := range raw {
		if i < len(raw)-1 {
			out = append(out, p+"\n\n")
		} else {
			out = append(out, p)
		}
	}
	return out
}

// ParagraphStableCount returns how many complete paragraphs exist in prose (for tests).
func ParagraphStableCount(content string) int {
	parts := splitByFences(content)
	if len(parts) == 0 {
		return 0
	}
	tail := parts[len(parts)-1]
	if tail.fenced || hasUnclosedFence(tail.text) {
		return 0
	}
	paras := splitParagraphs(tail.text)
	if len(paras) <= 1 {
		return 0
	}
	return len(paras) - 1
}

// StableFencePartCount returns complete fence segments before the tail (for tests).
func StableFencePartCount(content string) int {
	parts := splitByFences(content)
	if len(parts) <= 1 {
		return 0
	}
	return len(parts) - 1
}

// SegmentCacheLen returns cached segment count (for tests).
func (c *SegmentCache) SegmentCacheLen() int {
	if c == nil {
		return 0
	}
	return len(c.stableParts)
}

// SegmentCacheKeyPrefix returns whether a key with prefix exists (for tests).
func (c *SegmentCache) SegmentCacheKeyPrefix(prefix string) bool {
	for _, e := range c.stableParts {
		if strings.HasPrefix(e.key, prefix) {
			return true
		}
	}
	return false
}

// FencePartIndex is exported for tests to inspect split boundaries.
func FencePartIndex(content string, idx int) (fenced bool, size int) {
	parts := splitByFences(content)
	if idx < 0 || idx >= len(parts) {
		return false, 0
	}
	p := parts[idx]
	if p.fenced {
		return true, len(p.code)
	}
	return false, len(p.text)
}

// FormatSegmentKey is a test helper.
func FormatSegmentKey(width int, n int) string {
	return "p:" + strconv.Itoa(width) + ":" + strconv.Itoa(n)
}
