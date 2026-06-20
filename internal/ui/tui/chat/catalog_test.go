package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func catalogInput(header string, blocks []Block) CatalogInput {
	return CatalogInput{
		Header: header,
		Blocks: blocks,
		Width:  80,
		Now:    time.Now(),
		Disp:   tool.DisplayContext{},
	}
}

func TestLineCatalog_totalLines(t *testing.T) {
	var c LineCatalog
	c.Rebuild(catalogInput("header", nil))
	if c.TotalLines() != 1 {
		t.Fatalf("totalLines = %d, want 1", c.TotalLines())
	}
}

func TestLineCatalog_windowCost(t *testing.T) {
	blocks500 := []Block{{Role: RoleUser, Content: strings.Repeat("x\n", 500)}}
	blocks1000 := []Block{{Role: RoleUser, Content: strings.Repeat("x\n", 1000)}}

	var c500, c1000 LineCatalog
	in500 := catalogInput("", blocks500)
	in1000 := catalogInput("", blocks1000)
	c500.Rebuild(in500)
	c1000.Rebuild(in1000)

	start500 := len(c500.VisibleStyled(0, 20, in500))
	start1000 := len(c1000.VisibleStyled(0, 20, in1000))
	if start500 == 0 || start1000 == 0 {
		t.Fatal("expected visible window lines")
	}
	if start1000 > start500*2 {
		t.Fatalf("visible window should not scale linearly with total lines: %d vs %d", start1000, start500)
	}
}

func TestLineCatalog_rebuildCost(t *testing.T) {
	blocks500 := []Block{{Role: RoleUser, Content: strings.Repeat("line\n", 500)}}
	blocks1000 := []Block{{Role: RoleUser, Content: strings.Repeat("line\n", 1000)}}

	var c500, c1000 LineCatalog
	in500 := catalogInput("", blocks500)
	in1000 := catalogInput("", blocks1000)

	start := time.Now()
	c500.Rebuild(in500)
	t500 := time.Since(start)

	start = time.Now()
	c1000.Rebuild(in1000)
	t1000 := time.Since(start)

	if t1000 > t500*3 {
		t.Fatalf("rebuild scaled too much: 500=%v 1000=%v", t500, t1000)
	}
}

func TestLineCatalog_incrementalTail(t *testing.T) {
	var c LineCatalog
	var cache RenderCache
	blocks := []Block{{Role: RoleUser, Content: "hello"}}
	in := catalogInput("hdr", blocks)
	in.Cache = &cache
	c.Rebuild(in)
	before := len(c.plain)

	blocks = append(blocks, Block{Role: RoleUser, Content: "world"})
	in.Blocks = blocks
	c.Rebuild(in)
	if len(c.plain) <= before {
		t.Fatalf("plain lines = %d, want growth from %d", len(c.plain), before)
	}
}

func TestLineCatalog_noStyledStorage(t *testing.T) {
	var c LineCatalog
	blocks := []Block{{Role: RoleUser, Content: strings.Repeat("x\n", 200)}}
	in := catalogInput("", blocks)
	c.Rebuild(in)
	// VisibleStyled must work without a full styled slice on the catalog.
	visible := c.VisibleStyled(0, 10, in)
	if len(visible) == 0 {
		t.Fatal("expected visible styled window")
	}
}

func TestWindowSize_invalidatesCatalog(t *testing.T) {
	var c LineCatalog
	blocks := []Block{{Role: RoleUser, Content: strings.Repeat("word ", 200)}}
	wide := catalogInput("", blocks)
	wide.Width = 80
	c.Rebuild(wide)
	wideLines := c.TotalLines()

	narrow := catalogInput("", blocks)
	narrow.Width = 40
	c.Rebuild(narrow)
	narrowLines := c.TotalLines()

	if narrowLines <= wideLines {
		t.Fatalf("narrow width should increase line count: wide=%d narrow=%d", wideLines, narrowLines)
	}
	if !c.NeedsRebuild(80, false) {
		t.Fatal("catalog should need rebuild after width change")
	}
}

func TestVirtualList_streamTailInvalidate(t *testing.T) {
	var c LineCatalog
	var cache RenderCache
	blocks := []Block{{Role: RoleUser, Content: "hello"}}
	in := catalogInput("hdr", blocks)
	in.Cache = &cache
	c.Rebuild(in)
	prefix := append([]string(nil), c.plain[:min(3, len(c.plain))]...)

	blocks = append(blocks, Block{Role: RoleUser, Content: "world"})
	in.Blocks = blocks
	c.Rebuild(in)
	if len(c.plain) <= len(prefix) {
		t.Fatal("expected tail append to grow catalog")
	}
	for i := range prefix {
		if c.plain[i] != prefix[i] {
			t.Fatalf("prefix line %d changed on tail invalidate", i)
		}
	}
}
