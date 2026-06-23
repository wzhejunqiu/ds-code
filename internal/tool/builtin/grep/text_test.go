package grep

import (
	"strings"
	"testing"
)

func TestRenderDesc_noTemplateLeaks(t *testing.T) {
	desc := RenderDesc()
	if desc == "" {
		t.Fatal("empty description")
	}
	if strings.Contains(desc, "{{.") {
		t.Fatalf("unrendered template in description: %q", desc)
	}
}

func TestRenderDesc_noGitignoreMention(t *testing.T) {
	desc := RenderDesc()
	if strings.Contains(strings.ToLower(desc), "gitignore") {
		t.Fatalf("description must not mention gitignore: %q", desc)
	}
}
