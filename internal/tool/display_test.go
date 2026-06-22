package tool_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
)

func TestDisplaySummary_shell(t *testing.T) {
	args := `{"description":"Run tests","command":"cd /tmp && go test ./..."}`
	line, cmd := tool.DisplaySummary("bash", []byte(args), "", tool.DisplayContext{})
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
	args := `{"filepath":"foo.go","offset":10,"limit":11}`
	line, cmd := tool.DisplaySummary("read_file", []byte(args), "", tool.DisplayContext{})
	if cmd != "" {
		t.Fatalf("unexpected command: %q", cmd)
	}
	if line != "Read foo.go" {
		t.Fatalf("line = %q", line)
	}
}

func TestDisplaySummary_grepGlobList(t *testing.T) {
	ws := "/Users/me/ds-code"
	line, _ := tool.DisplaySummary("grep", []byte(`{"pattern":"package","path":"."}`), ws, tool.DisplayContext{})
	if line != "Grepped package in ds-code" {
		t.Fatalf("grep = %q", line)
	}
	line, _ = tool.DisplaySummary("glob", []byte(`{"pattern":"**/*.go","path":"internal/tuitest"}`), ws, tool.DisplayContext{})
	if line != "Searched files **/*.go in tuitest" {
		t.Fatalf("glob = %q", line)
	}
	line, _ = tool.DisplaySummary("list_dir", []byte(`{"path":""}`), ws, tool.DisplayContext{})
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

func TestDisplay_MCPBareName(t *testing.T) {
	reg := tool.NewRegistry()
	reg.RegisterMCPTool(&stubMCPTool{name: "semantic_search_nodes"}, "code-review-graph")
	disp := tool.FromRegistry(reg)
	line, cmd := tool.DisplaySummary("semantic_search_nodes", []byte(`{"query":"x"}`), "", disp)
	if cmd != "" {
		t.Fatalf("cmd = %q", cmd)
	}
	if line != "MCP code-review-graph · semantic_search_nodes {\"query\":\"x\"}" {
		t.Fatalf("line = %q", line)
	}
	if !tool.UsesHumanDisplay("semantic_search_nodes", disp) {
		t.Fatal("bare MCP should use human display")
	}
}

func TestDisplay_LegacyMCPPrefix(t *testing.T) {
	line, _ := tool.DisplaySummary("mcp__fs__read_file", []byte(`{"x":1}`), "", tool.DisplayContext{})
	if line != "MCP fs · read_file {\"x\":1}" {
		t.Fatalf("line = %q", line)
	}
	if !tool.UsesHumanDisplay("mcp__fs__read_file", tool.DisplayContext{}) {
		t.Fatal("legacy MCP prefix should use human display")
	}
}

func TestFormatMCPCallDisplay_emptyArgs(t *testing.T) {
	if got := tool.FormatMCPCallDisplay("srv", "tool", []byte(`{}`)); got != "MCP srv · tool" {
		t.Fatalf("empty {} should not append JSON: %q", got)
	}
	if got := tool.FormatMCPCallDisplay("srv", "tool", []byte(`null`)); got != "MCP srv · tool" {
		t.Fatalf("null args should not append JSON: %q", got)
	}
	if got := tool.FormatMCPCallDisplay("srv", "tool", nil); got != "MCP srv · tool" {
		t.Fatalf("nil args should not append JSON: %q", got)
	}
}

type stubMCPTool struct{ name string }

func (s *stubMCPTool) Name() string        { return s.name }
func (s *stubMCPTool) Description() string { return "mcp" }
func (s *stubMCPTool) Schema() map[string]any {
	return tool.ObjectSchema(nil, nil, false)
}
func (s *stubMCPTool) Execute(context.Context, json.RawMessage) (string, error) {
	return "", nil
}
func (s *stubMCPTool) PermissionLevel() permission.Level { return permission.LevelLow }

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
	t.Run("files_with_matches_no_match", func(t *testing.T) {
		args := []byte(`{"pattern":"foo","path":"."}`)
		line := tool.AppendGrepResultSuffix("Grepped foo in bar", args, builtin.ResultGrepNoMatches)
		if line != "Grepped foo in bar · 0 paths" {
			t.Fatalf("got %q", line)
		}
	})
	t.Run("files_with_matches_paths", func(t *testing.T) {
		args := []byte(`{"pattern":"foo"}`)
		result := "a.go\nb.go\n... 已截断，共 2 个文件"
		line := tool.AppendGrepResultSuffix("Grepped foo in bar", args, result)
		if line != "Grepped foo in bar · 2 paths" {
			t.Fatalf("got %q", line)
		}
	})
	t.Run("content_matches", func(t *testing.T) {
		args := []byte(`{"pattern":"foo","output_mode":"content"}`)
		result := "a.go:1:foo\na.go:2:foo"
		line := tool.AppendGrepResultSuffix("Grepped foo in bar", args, result)
		if line != "Grepped foo in bar · 2 matches" {
			t.Fatalf("got %q", line)
		}
	})
	t.Run("count", func(t *testing.T) {
		args := []byte(`{"pattern":"foo","output_mode":"count"}`)
		line := tool.AppendGrepResultSuffix("Grepped foo in bar", args, "42")
		if line != "Grepped foo in bar · 42 matches" {
			t.Fatalf("got %q", line)
		}
	})
	t.Run("count_zero", func(t *testing.T) {
		args := []byte(`{"pattern":"foo","output_mode":"count"}`)
		line := tool.AppendGrepResultSuffix("Grepped foo in bar", args, "0")
		if line != "Grepped foo in bar · 0 matches" {
			t.Fatalf("got %q", line)
		}
	})
}
