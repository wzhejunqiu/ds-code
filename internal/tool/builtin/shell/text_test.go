package shell

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestRenderDesc_injectsBuiltinToolNames(t *testing.T) {
	desc := RenderDesc()
	for _, want := range []string{
		tool.NameReadFile.String(),
		tool.NameGrep.String(),
		tool.NameGlob.String(),
		tool.NameListDir.String(),
		tool.NameApplyPatch.String(),
		tool.NameWriteFile.String(),
		tool.NameShell.String(),
	} {
		if !strings.Contains(desc, want) {
			t.Fatalf("missing tool name %q in RenderDesc", want)
		}
	}
	if strings.Contains(desc, "{{.") {
		t.Fatalf("unexpanded template placeholder in RenderDesc")
	}
	if !strings.Contains(desc, "run_in_background") {
		t.Fatalf("missing run_in_background job guidance")
	}
}

func TestShellTool_Description_matchesRenderDesc(t *testing.T) {
	if (&ShellTool{}).Description() != RenderDesc() {
		t.Fatal("ShellTool.Description should delegate to RenderDesc")
	}
}
