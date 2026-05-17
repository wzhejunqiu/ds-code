package tui

import "testing"

func TestGlamourStress(t *testing.T) {
	content := "# 📋 Title\n\n**bold** `code`\n\n- list\n\n```go\npanic(\"x\")\n```\n"
	for w := 1; w < 120; w++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("width %d: %v", w, r)
				}
			}()
			if _, err := renderMarkdown(content, w); err != nil {
				t.Fatalf("width %d err: %v", w, err)
			}
			_ = renderAssistantBlock(content, w)
		}()
	}
}
