package chattool

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func renderOut(b Block, width int, showDetails bool) string {
	return strings.Join(Render(b, width, showDetails, tool.DisplayContext{}), "\n")
}

func TestRenderBlockCollapsed_wrapsAtWidth(t *testing.T) {
	long := strings.Repeat("x", 120)
	out := renderOut(Block{
		Name: "bash", Command: "echo",
		Result: long + "\n",
	}, 40, false)
	if strings.Contains(out, long) {
		t.Fatal("collapsed result should wrap at terminal width")
	}
}

func TestRenderBlockCollapsed(t *testing.T) {
	out := renderOut(Block{
		Name: "bash", Args: "echo hi",
		Command: "echo\x00echo hi", Result: "hi\n",
	}, 60, false)
	for _, want := range []string{"echo hi", "echo", "└", "hi"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	for _, absent := range []string{"args:", "result:", "ctrl+o"} {
		if strings.Contains(out, absent) {
			t.Fatalf("collapsed view should not show %q:\n%s", absent, out)
		}
	}
}

func TestBuildResultPreview(t *testing.T) {
	got := buildResultPreview("one\ntwo\nthree\nfour")
	if len(got.lines) != 3 || got.lines[0] != "one" || got.moreLines != 1 {
		t.Fatalf("preview = %+v", got)
	}
	longLine := strings.Repeat("x", 300)
	got = buildResultPreview(longLine)
	if len(got.lines) != 1 || len(got.lines[0]) > resultPreviewMax {
		t.Fatalf("preview = %+v", got)
	}
	if !got.truncated || !strings.HasSuffix(got.lines[0], "...") {
		t.Fatalf("long line should be truncated: %+v", got)
	}
}

func TestRenderBlockExpandHint(t *testing.T) {
	var lines []string
	for i := 1; i <= 6; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	out := renderOut(Block{
		Name: "bash", Command: "seq",
		Result: strings.Join(lines, "\n"),
	}, 60, false)
	if !strings.Contains(out, "+3 lines (ctrl+o to expand)") {
		t.Fatalf("missing expand hint:\n%s", out)
	}
}

func TestRenderBlockExpanded(t *testing.T) {
	out := renderOut(Block{
		Name: "bash", Args: "echo hi",
		Command: "echo\x00echo hi", Result: "hi\n",
	}, 60, true)
	for _, want := range []string{"echo hi", "command:", "echo hi", "└", "hi"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "args:") {
		t.Fatalf("shell should not repeat args:\n%s", out)
	}
}
