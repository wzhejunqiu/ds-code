package toolname_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/toolname"
)

func TestBash_matchesToolNameShell(t *testing.T) {
	if tool.NameShell.String() != toolname.Bash {
		t.Fatalf("tool.NameShell=%q toolname.Bash=%q", tool.NameShell, toolname.Bash)
	}
}
