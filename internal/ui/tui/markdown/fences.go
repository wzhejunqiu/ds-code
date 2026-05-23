package markdown

import (
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
)

type part struct {
	text   string
	fenced bool
	lang   string
	code   string
}

func splitByFences(content string) []part {
	if content == "" {
		return nil
	}
	var parts []part
	rest := content
	for {
		start := strings.Index(rest, "```")
		if start < 0 {
			if rest != "" {
				parts = append(parts, part{text: rest})
			}
			break
		}
		if start > 0 {
			parts = append(parts, part{text: rest[:start]})
		}
		afterOpen := rest[start+3:]
		langLine, codeBody, ok := strings.Cut(afterOpen, "\n")
		if !ok {
			parts = append(parts, part{text: rest[start:]})
			break
		}
		lang := strings.TrimSpace(langLine)
		closeAt := findClosingFenceLine(codeBody)
		if closeAt < 0 {
			parts = append(parts, part{text: rest[start:]})
			break
		}
		code := strings.TrimSuffix(codeBody[:closeAt], "\n")
		parts = append(parts, part{fenced: true, lang: lang, code: code})
		rest = codeBody[closeAt:]
		if afterClose, ok := strings.CutPrefix(rest, "```"); ok {
			rest = afterClose
			if nl := strings.Index(rest, "\n"); nl >= 0 {
				rest = rest[nl+1:]
			} else {
				rest = ""
			}
		}
	}
	return parts
}

func findClosingFenceLine(s string) int {
	offset := 0
	for {
		nl := strings.Index(s[offset:], "\n")
		var line string
		lineStart := offset
		if nl < 0 {
			line = s[offset:]
			offset = len(s)
		} else {
			line = s[offset : offset+nl]
			offset += nl + 1
		}
		if strings.TrimSpace(line) == "```" {
			return lineStart
		}
		if nl < 0 {
			break
		}
	}
	return -1
}

func fencedMarkdown(lang, code string) string {
	if lang == "" {
		return "```\n" + code + "\n```"
	}
	return "```" + lang + "\n" + code + "\n```"
}

func codeBlockInnerWidth(outer int) int {
	const frame = 4
	if outer <= frame {
		return 1
	}
	return outer - frame
}

func boxRenderedCodeBlock(rendered string) string {
	lines := trimEmptyLines(strings.Split(strings.TrimRight(rendered, "\n"), "\n"))
	if len(lines) == 0 {
		return ""
	}
	return codeBlockBoxStyle.Render(strings.Join(lines, "\n")) + "\n"
}

func trimEmptyLines(lines []string) []string {
	out := lines[:0]
	for _, line := range lines {
		if strings.TrimSpace(StripANSI(line)) == "" {
			continue
		}
		out = append(out, strings.TrimRight(line, " "))
	}
	return out
}

func normalizeOutput(s string) string {
	s = strings.TrimLeft(s, "\n")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

// StripANSI removes ANSI escape sequences from s.
func StripANSI(s string) string {
	return xansi.Strip(s)
}
