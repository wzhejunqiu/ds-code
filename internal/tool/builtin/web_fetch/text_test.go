package web_fetch_test

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/web_fetch"
)

func TestRenderDesc_noTemplateLeak(t *testing.T) {
	desc := web_fetch.RenderDesc()
	if strings.Contains(desc, "{{.") {
		t.Fatalf("unrendered template in description: %q", desc)
	}
	if !strings.Contains(desc, "web_fetch") {
		t.Fatalf("expected tool name in description: %q", desc)
	}
	if !strings.Contains(desc, "prompt") {
		t.Fatalf("expected prompt in description: %q", desc)
	}
}
