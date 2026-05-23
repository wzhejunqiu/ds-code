package tool_test

import (
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
)

func TestDisplaySummary_shell(t *testing.T) {
	args := `{"description":"Run tests","command":"cd /tmp && go test ./..."}`
	line, cmd := tool.DisplaySummary("shell", []byte(args), "")
	if line != "Run tests" {
		t.Fatalf("argsLine = %q", line)
	}
	if tool.ShellCommandsList(cmd) != "cd, go" {
		t.Fatalf("commands = %q", tool.ShellCommandsList(cmd))
	}
	if tool.ShellFullCommand(cmd) != "cd /tmp && go test ./..." {
		t.Fatalf("full = %q", tool.ShellFullCommand(cmd))
	}
}

func TestDisplaySummary_readFile(t *testing.T) {
	args := `{"path":"foo.go","offset":10,"limit":11}`
	line, cmd := tool.DisplaySummary("read_file", []byte(args), "")
	if cmd != "" {
		t.Fatalf("unexpected command: %q", cmd)
	}
	if line != "Read foo.go" {
		t.Fatalf("line = %q", line)
	}
}

func TestDisplaySummary_grepGlobList(t *testing.T) {
	ws := "/Users/me/ds-code"
	line, _ := tool.DisplaySummary("grep", []byte(`{"pattern":"package","path":"."}`), ws)
	if line != "Grepped package in ds-code" {
		t.Fatalf("grep = %q", line)
	}
	line, _ = tool.DisplaySummary("glob", []byte(`{"pattern":"**/*.go","path":"internal/tuitest"}`), ws)
	if line != "Searched files **/*.go in tuitest" {
		t.Fatalf("glob = %q", line)
	}
	line, _ = tool.DisplaySummary("list_dir", []byte(`{"path":""}`), ws)
	if line != "List ds-code" {
		t.Fatalf("list_dir = %q", line)
	}
}

func TestParseShellCommands(t *testing.T) {
	names := tool.ParseShellCommands("cd /tmp && go test ./... | head -40")
	if len(names) != 3 || names[0] != "cd" || names[1] != "go" || names[2] != "head" {
		t.Fatalf("names = %v", names)
	}
	if got := tool.FormatShellCommandsList(names); got != "cd, 2+" {
		t.Fatalf("list = %q", got)
	}
	if got := tool.FormatShellCommandsList([]string{"cd", "go", "head", "make"}); got != "cd, 3+" {
		t.Fatalf("list4 = %q", got)
	}
}

func TestApplyPatchFileDisplays(t *testing.T) {
	patch := `*** Begin Patch
*** Update File: a.go
@@
-old
+new
*** Update File: b.go
@@
-removed
*** End Patch`
	displays := tool.ApplyPatchFileDisplays(patch, "")
	if len(displays) != 2 {
		t.Fatalf("got %d displays", len(displays))
	}
	if displays[0].Filename != "a.go" || displays[0].Added != 1 || displays[0].Removed != 1 {
		t.Fatalf("a.go = %+v", displays[0])
	}
	if displays[1].Filename != "b.go" || displays[1].Removed != 1 {
		t.Fatalf("b.go = %+v", displays[1])
	}
}

func TestFormatMCPDisplay(t *testing.T) {
	if got := tool.FormatMCPDisplay("mcp__fs__read_file"); got != "MCP fs · read_file" {
		t.Fatalf("got %q", got)
	}
}

func TestReadFileLineRange(t *testing.T) {
	result := "9|nine\n10|ten\n20|twenty"
	start, end, ok := tool.ReadFileLineRange(result)
	if !ok || start != 9 || end != 20 {
		t.Fatalf("range = %d-%d ok=%v", start, end, ok)
	}
	got := tool.AppendReadFileLineRange("Read foo.go", start, end)
	if got != "Read foo.go L9-20" {
		t.Fatalf("got %q", got)
	}
}

func TestAppendGrepResultSuffix(t *testing.T) {
	line := tool.AppendGrepResultSuffix("Grepped foo in bar", builtin.ResultGrepNoMatches)
	if !strings.Contains(line, "0 matches") {
		t.Fatalf("got %q", line)
	}
}
