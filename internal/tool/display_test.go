package tool_test

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/tool"
)

func TestDisplaySummary_shell(t *testing.T) {
	args := `{"command":"ls -la"}`
	line, cmd := tool.DisplaySummary("shell", []byte(args))
	if cmd != "ls -la" {
		t.Fatalf("command = %q", cmd)
	}
	if line == "" {
		t.Fatal("expected args line")
	}
}

func TestDisplaySummary_readFile(t *testing.T) {
	args := `{"path":"foo.go","offset":10,"limit":20}`
	line, cmd := tool.DisplaySummary("read_file", []byte(args))
	if cmd != "" {
		t.Fatalf("unexpected command: %q", cmd)
	}
	if line != "path=foo.go, offset=10, limit=20" {
		t.Fatalf("line = %q", line)
	}
}
