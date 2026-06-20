package chat

import (
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/markdown"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/selection"
)

// LineCatalog holds the full styled/plain line index for virtual viewport rendering.
type LineCatalog struct {
	styled  []string
	plain   []string
	width   int
	details bool
}

// Reset clears the catalog.
func (c *LineCatalog) Reset() {
	if c == nil {
		return
	}
	c.styled = nil
	c.plain = nil
	c.width = 0
	c.details = false
}

// TotalLines returns the number of lines in the catalog.
func (c *LineCatalog) TotalLines() int {
	if c == nil {
		return 0
	}
	return len(c.styled)
}

// PlainLines returns the full plain-text lines for selection.
func (c *LineCatalog) PlainLines() []string {
	if c == nil {
		return nil
	}
	return c.plain
}

// StyledLines returns the full styled lines.
func (c *LineCatalog) StyledLines() []string {
	if c == nil {
		return nil
	}
	return c.styled
}

// VisibleStyled returns styled lines for [yOffset, yOffset+height).
func (c *LineCatalog) VisibleStyled(yOffset, height int) []string {
	if c == nil || height <= 0 || len(c.styled) == 0 {
		return nil
	}
	end := yOffset + height
	if end > len(c.styled) {
		end = len(c.styled)
	}
	if yOffset >= len(c.styled) {
		return nil
	}
	return c.styled[yOffset:end]
}

// Rebuild reconstructs the catalog from chat blocks and header text.
func (c *LineCatalog) Rebuild(header string, blocks []Block, width int, now time.Time, showToolDetails bool, disp tool.DisplayContext, cache *RenderCache, mdCache *markdown.SegmentCache) {
	if c == nil {
		return
	}
	if width < 10 {
		width = 10
	}
	body := RenderCachedLines(blocks, width, now, showToolDetails, disp, cache, mdCache)
	var lines []string
	if header != "" {
		lines = append(lines, strings.Split(header, "\n")...)
		if len(body) > 0 {
			lines = append(lines, "", "")
		}
	}
	lines = append(lines, body...)

	c.styled = lines
	c.plain = make([]string, len(lines))
	for i, line := range lines {
		c.plain[i] = selection.StripANSI(line)
	}
	c.width = width
	c.details = showToolDetails
}

// NeedsRebuild reports whether width or tool-details changed since last rebuild.
func (c *LineCatalog) NeedsRebuild(width int, showToolDetails bool) bool {
	if c == nil {
		return true
	}
	return c.width != width || c.details != showToolDetails
}
