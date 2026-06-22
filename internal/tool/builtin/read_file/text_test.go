package read_file

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestRenderDesc_injectsBuiltinToolNames(t *testing.T) {
	desc := RenderDesc()
	if !strings.Contains(desc, tool.NameShell.String()) {
		t.Fatalf("missing shell tool name in RenderDesc: %q", desc)
	}
	if strings.Contains(desc, "{{.") {
		t.Fatalf("unexpanded template placeholder in RenderDesc: %q", desc)
	}
}

func TestReadFileTool_Description_matchesRenderDesc(t *testing.T) {
	if (&ReadFileTool{}).Description() != RenderDesc() {
		t.Fatal("ReadFileTool.Description should delegate to RenderDesc")
	}
}
