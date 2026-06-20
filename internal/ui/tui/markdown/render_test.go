package markdown

import (
	"os"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestRenderHeadingsStyled(t *testing.T) {
	content := "# Title\n\n## Subtitle\n\n### Section\n\nbody"
	out, err := Render(content, 60)
	if err != nil {
		t.Fatal(err)
	}
	plain := StripANSI(out)
	for _, marker := range []string{"# ", "## ", "### ", "#### "} {
		if strings.Contains(plain, marker) {
			t.Fatalf("raw heading marker %q in output:\n%s", marker, plain)
		}
	}
	for _, text := range []string{"Title", "Subtitle", "Section", "body"} {
		if !headingTextPresent(out, text) {
			t.Fatalf("missing %q in output:\n%s", text, out)
		}
	}
}

func TestMarkdownRender_colorProfile(t *testing.T) {
	old := lipgloss.Writer.Profile
	lipgloss.Writer.Profile = colorprofile.Ascii
	t.Cleanup(func() { lipgloss.Writer.Profile = old })

	out, err := Render("# Title\n\n**bold** body", 60)
	if err != nil {
		t.Fatal(err)
	}
	plain := StripANSI(out)
	if !strings.Contains(plain, "Title") || !strings.Contains(plain, "body") {
		t.Fatalf("unexpected render under Ascii profile:\n%s", plain)
	}
}

func TestSplitByFences_codeWithInlineBackticks(t *testing.T) {
	content := "before\n\n```go\ns := \"```not a fence\"\nfmt.Println(1)\n```\n\nafter"
	parts := splitByFences(content)
	if len(parts) != 3 {
		t.Fatalf("parts = %d, want 3", len(parts))
	}
	if !parts[1].fenced || !strings.Contains(parts[1].code, "not a fence") {
		t.Fatalf("unexpected fenced part: %+v", parts[1])
	}
}

func headingTextPresent(out, text string) bool {
	plain := strings.ReplaceAll(StripANSI(out), " ", "")
	compact := strings.ReplaceAll(text, " ", "")
	return strings.Contains(plain, compact)
}
