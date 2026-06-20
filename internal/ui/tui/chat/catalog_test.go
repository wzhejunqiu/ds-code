package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestLineCatalog_totalLines(t *testing.T) {
	var c LineCatalog
	c.Rebuild("header", nil, 80, time.Now(), false, tool.DisplayContext{}, nil, nil)
	if c.TotalLines() != 1 {
		t.Fatalf("totalLines = %d, want 1", c.TotalLines())
	}
}

func TestLineCatalog_windowCost(t *testing.T) {
	blocks500 := make([]Block, 1)
	blocks500[0] = Block{Role: RoleUser, Content: strings.Repeat("x\n", 500)}
	blocks1000 := make([]Block, 1)
	blocks1000[0] = Block{Role: RoleUser, Content: strings.Repeat("x\n", 1000)}

	var c500, c1000 LineCatalog
	now := time.Now()
	disp := tool.DisplayContext{}
	c500.Rebuild("", blocks500, 80, now, false, disp, nil, nil)
	c1000.Rebuild("", blocks1000, 80, now, false, disp, nil, nil)

	start500 := len(c500.VisibleStyled(0, 20))
	start1000 := len(c1000.VisibleStyled(0, 20))
	if start500 == 0 || start1000 == 0 {
		t.Fatal("expected visible window lines")
	}
	if start1000 > start500*2 {
		t.Fatalf("visible window should not scale linearly with total lines: %d vs %d", start1000, start500)
	}
}
