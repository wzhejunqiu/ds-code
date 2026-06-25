package agent

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestRenderDesc_noTemplateLeaks(t *testing.T) {
	desc := RenderDesc(3)
	if desc == "" {
		t.Fatal("empty description")
	}
	if strings.Contains(desc, "{{.") {
		t.Fatalf("unrendered template in description: %q", desc)
	}
}

func TestRenderDesc_containsKeySections(t *testing.T) {
	desc := RenderDesc(5)
	for _, want := range []string{
		"general-purpose",
		"Explore",
		"提示词撰写规范",
		"使用须知",
		tool.NameAgent.String(),
		tool.NameReadFile.String(),
		tool.NameGlob.String(),
	} {
		if !strings.Contains(desc, want) {
			t.Fatalf("description missing %q", want)
		}
	}
	for _, absent := range []string{
		"Plan",
		"verification",
		"max_parallel",
		"Fork",
		"worktree",
	} {
		if strings.Contains(desc, absent) {
			t.Fatalf("description should not mention %q", absent)
		}
	}
}
