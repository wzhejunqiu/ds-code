package write_file

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestRenderDesc_injectsBuiltinToolNames(t *testing.T) {
	desc := RenderDesc()
	for _, name := range []string{
		tool.NameReadFile.String(),
		tool.NameApplyPatch.String(),
	} {
		if !strings.Contains(desc, name) {
			t.Fatalf("missing %q in RenderDesc: %q", name, desc)
		}
	}
	if strings.Contains(desc, "{{.") {
		t.Fatalf("unexpanded template placeholder in RenderDesc: %q", desc)
	}
}

func TestErrFmt_injectsBuiltinToolNames(t *testing.T) {
	for name, fmtStr := range map[string]string{
		"ErrMustReadFirstFmt":      ErrMustReadFirstFmt,
		"ErrSameBatchReadWriteFmt": ErrSameBatchReadWriteFmt,
	} {
		if strings.Contains(fmtStr, "{{.") {
			t.Fatalf("unexpanded placeholder in %s: %q", name, fmtStr)
		}
		for _, toolName := range []string{
			tool.NameReadFile.String(),
			tool.NameWriteFile.String(),
		} {
			if !strings.Contains(fmtStr, toolName) {
				t.Fatalf("missing %q in %s: %q", toolName, name, fmtStr)
			}
		}
	}
}

func TestWriteFileTool_Description_matchesRenderDesc(t *testing.T) {
	if (&WriteFileTool{}).Description() != RenderDesc() {
		t.Fatal("WriteFileTool.Description should delegate to RenderDesc")
	}
}
