package chat

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chattool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/markdown"
)

const liveTimeBucketMs = 100

// RenderCache caches rendered lines per chat block index.
type RenderCache struct {
	width           int
	showToolDetails bool
	entries         []cacheEntry
	mdBlockIdx      int
}

type cacheEntry struct {
	key   string
	lines []string
}

// Reset clears all cached block renderings.
func (c *RenderCache) Reset() {
	if c == nil {
		return
	}
	c.entries = nil
	c.width = 0
	c.showToolDetails = false
	c.mdBlockIdx = -1
}

// RenderCached formats chat blocks, reusing cached lines when fingerprints match.
func RenderCached(blocks []Block, width int, now time.Time, showToolDetails bool, disp tool.DisplayContext, cache *RenderCache, mdCache *markdown.SegmentCache) string {
	if width < 20 {
		width = 20
	}
	if cache == nil {
		return renderAllBlocks(blocks, width, now, showToolDetails, disp, mdCache)
	}
	if cache.width != width || cache.showToolDetails != showToolDetails {
		cache.Reset()
		cache.width = width
		cache.showToolDetails = showToolDetails
	}
	if len(cache.entries) != len(blocks) {
		cache.entries = resizeEntries(cache.entries, len(blocks))
		cache.mdBlockIdx = -1
	}

	var lines []string
	for i := range blocks {
		key := blockFingerprint(&blocks[i], now, showToolDetails)
		if i < len(cache.entries) && cache.entries[i].key == key && cache.entries[i].lines != nil {
			lines = append(lines, cache.entries[i].lines...)
			continue
		}
		if useMDCache(&blocks[i]) {
			if cache.mdBlockIdx != i {
				if mdCache != nil {
					mdCache.Reset()
				}
				cache.mdBlockIdx = i
			}
		}
		blockLines := renderBlock(&blocks[i], width, now, showToolDetails, disp, mdCache)
		cache.entries[i] = cacheEntry{key: key, lines: blockLines}
		lines = append(lines, blockLines...)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

// BlockLines returns cached styled lines for block i after SyncBlockLineSlices.
func (c *RenderCache) BlockLines(i int) []string {
	if c == nil || i < 0 || i >= len(c.entries) {
		return nil
	}
	return c.entries[i].lines
}

// SyncBlockLineSlices updates the block cache and returns styled line slices per block.
func SyncBlockLineSlices(blocks []Block, width int, now time.Time, showToolDetails bool, disp tool.DisplayContext, cache *RenderCache, mdCache *markdown.SegmentCache) [][]string {
	if width < 20 {
		width = 20
	}
	if cache == nil {
		out := make([][]string, len(blocks))
		for i := range blocks {
			out[i] = renderBlock(&blocks[i], width, now, showToolDetails, disp, mdCache)
		}
		return out
	}
	if cache.width != width || cache.showToolDetails != showToolDetails {
		cache.Reset()
		cache.width = width
		cache.showToolDetails = showToolDetails
	}
	if len(cache.entries) != len(blocks) {
		cache.entries = resizeEntries(cache.entries, len(blocks))
		cache.mdBlockIdx = -1
	}

	out := make([][]string, len(blocks))
	for i := range blocks {
		key := blockFingerprint(&blocks[i], now, showToolDetails)
		if i < len(cache.entries) && cache.entries[i].key == key && cache.entries[i].lines != nil {
			out[i] = cache.entries[i].lines
			continue
		}
		if useMDCache(&blocks[i]) {
			if cache.mdBlockIdx != i {
				if mdCache != nil {
					mdCache.Reset()
				}
				cache.mdBlockIdx = i
			}
		}
		blockLines := renderBlock(&blocks[i], width, now, showToolDetails, disp, mdCache)
		cache.entries[i] = cacheEntry{key: key, lines: blockLines}
		out[i] = blockLines
	}
	return out
}

// RenderCachedLines formats chat blocks into styled line slices, reusing block cache.
func RenderCachedLines(blocks []Block, width int, now time.Time, showToolDetails bool, disp tool.DisplayContext, cache *RenderCache, mdCache *markdown.SegmentCache) []string {
	slices := SyncBlockLineSlices(blocks, width, now, showToolDetails, disp, cache, mdCache)
	var lines []string
	for _, slice := range slices {
		lines = append(lines, slice...)
	}
	if len(lines) > 0 {
		for len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
	}
	return lines
}

func renderAllBlocks(blocks []Block, width int, now time.Time, showToolDetails bool, disp tool.DisplayContext, mdCache *markdown.SegmentCache) string {
	var lines []string
	for i := range blocks {
		lines = append(lines, renderBlock(&blocks[i], width, now, showToolDetails, disp, mdCache)...)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func resizeEntries(entries []cacheEntry, n int) []cacheEntry {
	oldLen := len(entries)
	if cap(entries) >= n {
		entries = entries[:n]
	} else {
		out := make([]cacheEntry, n)
		copy(out, entries)
		entries = out
	}
	for i := oldLen; i < n; i++ {
		entries[i] = cacheEntry{}
	}
	return entries
}

func useMDCache(b *Block) bool {
	return b.Role == RoleAssistant && b.Content != "" && (b.Streaming || blockNeedsLiveNow(b))
}

func blockNeedsLiveNow(b *Block) bool {
	if b.Role == RolePlanning {
		return true
	}
	if b.Streaming {
		return true
	}
	if b.Role == RoleAssistant && !b.ReasoningStartedAt.IsZero() && b.ReasoningEndedAt.IsZero() {
		return true
	}
	return false
}

func blockFingerprint(b *Block, now time.Time, showToolDetails bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "r%d|", b.Role)
	sb.WriteString(b.Content)
	sb.WriteByte('|')
	sb.WriteString(b.Reasoning)
	sb.WriteByte('|')
	fmt.Fprintf(&sb, "ro%d|", boolInt(b.ReasoningOpen))
	if !b.ReasoningStartedAt.IsZero() {
		sb.WriteString(b.ReasoningStartedAt.Format(time.RFC3339Nano))
	}
	sb.WriteByte('|')
	if !b.ReasoningEndedAt.IsZero() {
		sb.WriteString(b.ReasoningEndedAt.Format(time.RFC3339Nano))
	}
	sb.WriteByte('|')
	if !b.PlanningStartedAt.IsZero() {
		sb.WriteString(b.PlanningStartedAt.Format(time.RFC3339Nano))
	}
	sb.WriteByte('|')
	fmt.Fprintf(&sb, "rd%d|td%d|st%d|", b.ReasoningDuration, b.TurnDuration, boolInt(b.Streaming))
	sb.WriteString(b.ToolName)
	sb.WriteByte('|')
	sb.WriteString(b.ToolArgs)
	sb.WriteByte('|')
	sb.WriteString(b.ToolCommand)
	sb.WriteByte('|')
	sb.WriteString(b.ToolResult)
	fmt.Fprintf(&sb, "|tr%d|te%d|tx%d|", boolInt(b.ToolRunning), boolInt(b.ToolError), boolInt(b.ToolExpanded))
	fmt.Fprintf(&sb, "td%d|", boolInt(showToolDetails))
	if blockNeedsLiveNow(b) {
		fmt.Fprintf(&sb, "t%d", now.UnixMilli()/liveTimeBucketMs)
	}
	return sb.String()
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func renderBlock(b *Block, width int, now time.Time, showToolDetails bool, disp tool.DisplayContext, mdCache *markdown.SegmentCache) []string {
	var lines []string
	switch b.Role {
	case RoleUser:
		lines = append(lines, renderUserBlock(b.Content, width)...)
		lines = append(lines, "")
	case RoleAssistant:
		indent := lipgloss.Width(assistantBullet)
		if b.Reasoning != "" || b.ReasoningDuration > 0 || !b.ReasoningStartedAt.IsZero() {
			expanded := reasoningExpanded(b)
			label := reasoningBlockLabel(expanded, b.ReasoningStartedAt, b.ReasoningEndedAt, now, b.ReasoningDuration)
			lines = append(lines, styleReason.Render(strings.Repeat(" ", indent)+label))
			if expanded && b.Reasoning != "" {
				lines = append(lines, styleReason.Render(markdown.WrapText(b.Reasoning, width-indent)))
			}
		}
		if b.Content != "" {
			lines = append(lines, renderAssistantBlock(b.Content, width, mdCache)...)
		} else if b.Streaming {
			lines = append(lines, renderAssistantLine(styleReason.Render("…")))
		}
		if b.TurnDuration > 0 {
			lines = append(lines, styleTurnMeta.Render(strings.Repeat(" ", indent)+turnDurationLine(b.TurnDuration)))
		}
		lines = append(lines, "")
	case RoleTool:
		lines = append(lines, chattool.Render(chattool.Block{
			Name: b.ToolName, Args: b.ToolArgs, Command: b.ToolCommand,
			Result: b.ToolResult, Running: b.ToolRunning, Error: b.ToolError,
			Expanded: b.ToolExpanded,
		}, width, showToolDetails, disp)...)
		lines = append(lines, "")
	case RolePlanning:
		indent := lipgloss.Width(planningBullet)
		lines = append(lines, styleReason.Render(strings.Repeat(" ", indent)+planningBlockLabel(b.PlanningStartedAt, now)))
		lines = append(lines, "")
	case RoleInterrupt:
		indent := lipgloss.Width(interruptBullet)
		lines = append(lines, styleInterrupt.Render(strings.Repeat(" ", indent)+interruptBullet+interruptLabel))
		lines = append(lines, "")
	}
	return lines
}
