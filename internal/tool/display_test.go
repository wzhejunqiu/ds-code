package tool_test

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/tool"
)

func TestDisplaySummary_shell(t *testing.T) {
	args := `{"command":"ls -la"}`
	line, cmd := tool.DisplaySummary("shell", []byte(args), "")
	if cmd != "ls -la" {
		t.Fatalf("command = %q", cmd)
	}
	if line == "" {
		t.Fatal("expected args line")
	}
}

func TestDisplaySummary_readFile(t *testing.T) {
	args := `{"path":"foo.go","start":10,"end":20}`
	line, cmd := tool.DisplaySummary("read_file", []byte(args), "")
	if cmd != "" {
		t.Fatalf("unexpected command: %q", cmd)
	}
	if line != "Read foo.go" {
		t.Fatalf("line = %q", line)
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
	if tool.FormatReadFileDisplay("sample.go", 1, 3) != "Read sample.go L1-3" {
		t.Fatal("FormatReadFileDisplay mismatch")
	}
}
