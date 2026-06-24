package glob_test

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/glob"
)

func TestRenderDesc_noTemplateLeak(t *testing.T) {
	desc := glob.RenderDesc()
	if strings.Contains(desc, "{{.") {
		t.Fatalf("unrendered template in description: %q", desc)
	}
	if !strings.Contains(desc, "glob") {
		t.Fatalf("expected tool name in description: %q", desc)
	}
}
