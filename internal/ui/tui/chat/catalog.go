package chat

import (
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/markdown"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/selection"
)

// CatalogInput carries render state for incremental line catalog updates.
type CatalogInput struct {
	Header          string
	Blocks          []Block
	Width           int
	Now             time.Time
	ShowToolDetails bool
	Disp            tool.DisplayContext
	Cache           *RenderCache
	MDCache         *markdown.SegmentCache
}

// LineCatalog holds plain lines for selection and block offsets for on-demand styled rendering.
// Styled lines are not stored in full; VisibleStyled renders only the requested window.
type LineCatalog struct {
	plain       []string
	headerKey   string
	headerLines int
	bodyStart   int
	blockStarts []int
	blockKeys   []string
	width       int
	details     bool
}

// Reset clears the catalog.
func (c *LineCatalog) Reset() {
	if c == nil {
		return
	}
	c.plain = nil
	c.headerKey = ""
	c.headerLines = 0
	c.bodyStart = 0
	c.blockStarts = nil
	c.blockKeys = nil
	c.width = 0
	c.details = false
}

// TotalLines returns the number of lines in the catalog.
func (c *LineCatalog) TotalLines() int {
	if c == nil {
		return 0
	}
	return len(c.plain)
}

// PlainLines returns the full plain-text lines for selection.
func (c *LineCatalog) PlainLines() []string {
	if c == nil {
		return nil
	}
	return c.plain
}

// VisibleStyled returns styled lines for [yOffset, yOffset+height) without materializing the full transcript.
func (c *LineCatalog) VisibleStyled(yOffset, height int, in CatalogInput) []string {
	if c == nil || height <= 0 || len(c.plain) == 0 || yOffset >= len(c.plain) {
		return nil
	}
	end := yOffset + height
	if end > len(c.plain) {
		end = len(c.plain)
	}
	out := make([]string, 0, end-yOffset)
	for i := yOffset; i < end; i++ {
		out = append(out, c.styledAt(i, in))
	}
	return out
}

// Rebuild updates plain lines and block metadata, reusing RenderCache and invalidating only from the first dirty block.
func (c *LineCatalog) Rebuild(in CatalogInput) {
	if c == nil {
		return
	}
	width := in.Width
	if width < 10 {
		width = 10
	}
	blockSlices := SyncBlockLineSlices(in.Blocks, width, in.Now, in.ShowToolDetails, in.Disp, in.Cache, in.MDCache)

	fullReset := c.NeedsRebuild(width, in.ShowToolDetails) || in.Header != c.headerKey
	firstDirty := 0
	if !fullReset {
		firstDirty = c.firstDirtyBlock(in, blockSlices)
		if firstDirty < 0 {
			return
		}
	}

	hdrLines := headerPlainLines(in.Header)
	bodyGap := headerBodyGap(in.Header, len(blockSlices))
	newStarts := computeBlockStarts(len(hdrLines), bodyGap, blockSlices)
	newKeys := blockKeys(in.Blocks, in.Now, in.ShowToolDetails)

	if fullReset || firstDirty == 0 {
		c.plain = assemblePlain(hdrLines, bodyGap, blockSlices)
	} else {
		truncate := 0
		if firstDirty < len(c.blockStarts) {
			truncate = c.blockStarts[firstDirty]
		} else if len(c.blockStarts) > 0 {
			truncate = len(c.plain)
		}
		c.plain = append(c.plain[:truncate:truncate], plainBlockTail(blockSlices, firstDirty)...)
	}

	c.headerKey = in.Header
	c.headerLines = len(hdrLines)
	c.bodyStart = len(hdrLines) + bodyGap
	c.blockStarts = newStarts
	c.blockKeys = newKeys
	c.width = width
	c.details = in.ShowToolDetails
}

// NeedsRebuild reports whether width or tool-details changed since last rebuild.
func (c *LineCatalog) NeedsRebuild(width int, showToolDetails bool) bool {
	if c == nil {
		return true
	}
	return c.width != width || c.details != showToolDetails
}

func (c *LineCatalog) firstDirtyBlock(in CatalogInput, slices [][]string) int {
	if in.Header != c.headerKey {
		return 0
	}
	keys := blockKeys(in.Blocks, in.Now, in.ShowToolDetails)
	if len(keys) != len(c.blockKeys) {
		if len(keys) > len(c.blockKeys) {
			return len(c.blockKeys)
		}
		return 0
	}
	for i := range keys {
		if keys[i] != c.blockKeys[i] {
			return i
		}
	}
	newStarts := computeBlockStarts(c.headerLines, headerBodyGap(in.Header, len(slices)), slices)
	if !blockStartsEqual(c.blockStarts, newStarts) {
		return 0
	}
	return -1
}

func (c *LineCatalog) styledAt(idx int, in CatalogInput) string {
	if idx < 0 || idx >= len(c.plain) {
		return ""
	}
	if c.headerLines > 0 && idx < c.headerLines {
		parts := strings.Split(in.Header, "\n")
		if idx < len(parts) {
			return parts[idx]
		}
		return ""
	}
	if idx < c.bodyStart {
		return ""
	}
	bi := blockIndexAt(c.blockStarts, idx)
	if bi < 0 || in.Cache == nil {
		return c.plain[idx]
	}
	off := idx - c.blockStarts[bi]
	if lines := in.Cache.BlockLines(bi); off < len(lines) {
		return lines[off]
	}
	return c.plain[idx]
}

func headerPlainLines(header string) []string {
	if header == "" {
		return nil
	}
	return strings.Split(header, "\n")
}

func headerBodyGap(header string, blockCount int) int {
	if header != "" && blockCount > 0 {
		return 2
	}
	return 0
}

func computeBlockStarts(headerLines, bodyGap int, blockSlices [][]string) []int {
	starts := make([]int, len(blockSlices))
	line := headerLines + bodyGap
	for i, slice := range blockSlices {
		starts[i] = line
		line += len(slice)
	}
	return starts
}

func assemblePlain(headerLines []string, bodyGap int, blockSlices [][]string) []string {
	n := len(headerLines) + bodyGap
	for _, slice := range blockSlices {
		n += len(slice)
	}
	out := make([]string, 0, n)
	for _, line := range headerLines {
		out = append(out, selection.StripANSI(line))
	}
	for i := 0; i < bodyGap; i++ {
		out = append(out, "")
	}
	for _, slice := range blockSlices {
		for _, line := range slice {
			out = append(out, selection.StripANSI(line))
		}
	}
	return out
}

func plainBlockTail(blockSlices [][]string, from int) []string {
	if from >= len(blockSlices) {
		return nil
	}
	var n int
	for i := from; i < len(blockSlices); i++ {
		n += len(blockSlices[i])
	}
	out := make([]string, 0, n)
	for i := from; i < len(blockSlices); i++ {
		for _, line := range blockSlices[i] {
			out = append(out, selection.StripANSI(line))
		}
	}
	return out
}

func blockKeys(blocks []Block, now time.Time, showToolDetails bool) []string {
	keys := make([]string, len(blocks))
	for i := range blocks {
		keys[i] = blockFingerprint(&blocks[i], now, showToolDetails)
	}
	return keys
}

func blockIndexAt(starts []int, idx int) int {
	bi := -1
	for i, start := range starts {
		if start <= idx {
			bi = i
		}
	}
	return bi
}

func blockStartsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
