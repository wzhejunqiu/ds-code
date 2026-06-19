package toolresult_test

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/toolresult"
)

func TestShortenSpillPathForHint_noTildeNoTruncate(t *testing.T) {
	abs := "/Users/me/.ds-code/projects/abc/mcp-result/sess/call_foo.txt"
	got := toolresult.ShortenSpillPathForHint(abs, 100)
	if got != abs {
		t.Fatalf("got %q want full path %q", got, abs)
	}
	if strings.Contains(got, "~") {
		t.Fatalf("hint path must not use tilde: %q", got)
	}
}

func TestMCPSavedResultHint_format(t *testing.T) {
	path := "/tmp/mcp-result/sess/call.txt"
	hint := toolresult.MCPSavedResultHint(path)
	if !strings.Contains(hint, path) {
		t.Fatalf("hint missing path: %q", hint)
	}
	if !strings.Contains(hint, "read_file") {
		t.Fatalf("hint should guide read_file: %q", hint)
	}
}
