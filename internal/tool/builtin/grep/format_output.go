package grep

import (
	"fmt"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
)

type grepPageMeta struct {
	Limit        int
	Offset       int
	TotalEntries int
	TotalFiles   int
	TotalMatches int
}

func formatPaginationFooter(limit, offset int) string {
	return fmt.Sprintf("[Showing results with pagination = limit: %d, offset: %d]", limit, offset)
}

func formatFilesSummary(totalFiles int) string {
	return fmt.Sprintf("Found %d files", totalFiles)
}

func formatCountSummary(totalMatches, totalFiles int) string {
	return fmt.Sprintf("Found %d occurrences across %d files", totalMatches, totalFiles)
}

func needsPaginationFooter(meta grepPageMeta, shown int) bool {
	if meta.Offset > 0 {
		return true
	}
	if meta.Limit > 0 && meta.TotalEntries > meta.Offset+shown {
		return true
	}
	return false
}

func formatGrepOutput(mode string, body []string, meta grepPageMeta) string {
	switch mode {
	case builtin.GrepOutputContent:
		return formatContentOutput(body, meta)
	case builtin.GrepOutputCount:
		return formatCountOutput(body, meta)
	default:
		return formatFilesOutput(body, meta)
	}
}

func formatFilesOutput(paths []string, meta grepPageMeta) string {
	if meta.TotalFiles == 0 {
		return formatFilesSummary(0)
	}
	var b strings.Builder
	b.WriteString(formatFilesSummary(meta.TotalFiles))
	for _, p := range paths {
		b.WriteByte('\n')
		b.WriteString(p)
	}
	if needsPaginationFooter(meta, len(paths)) {
		b.WriteByte('\n')
		b.WriteString(formatPaginationFooter(meta.Limit, meta.Offset))
	}
	return b.String()
}

func formatContentOutput(lines []string, meta grepPageMeta) string {
	if meta.TotalEntries == 0 {
		return ""
	}
	out := strings.Join(lines, "\n")
	if needsPaginationFooter(meta, len(lines)) {
		if out != "" {
			out += "\n"
		}
		out += formatPaginationFooter(meta.Limit, meta.Offset)
	}
	return out
}

func formatCountOutput(countLines []string, meta grepPageMeta) string {
	if meta.TotalFiles == 0 {
		return formatCountSummary(0, 0)
	}
	var b strings.Builder
	for i, line := range countLines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	if len(countLines) > 0 {
		b.WriteByte('\n')
	}
	b.WriteString(formatCountSummary(meta.TotalMatches, meta.TotalFiles))
	if needsPaginationFooter(meta, len(countLines)) {
		b.WriteByte('\n')
		b.WriteString(formatPaginationFooter(meta.Limit, meta.Offset))
	}
	return b.String()
}

func formatGrepLine(kind recordKind, rel string, line int, text string, showLineNums bool) string {
	text = strings.TrimSpace(text)
	switch kind {
	case recordContext:
		if showLineNums {
			return fmt.Sprintf("%s:%d-%s", rel, line, text)
		}
		return fmt.Sprintf("%s-%s", rel, text)
	default:
		if showLineNums {
			return fmt.Sprintf("%s:%d:%s", rel, line, text)
		}
		return fmt.Sprintf("%s:%s", rel, text)
	}
}

func emptyOutputForMode(mode string) string {
	switch mode {
	case builtin.GrepOutputContent:
		return ""
	case builtin.GrepOutputCount:
		return formatCountSummary(0, 0)
	default:
		return formatFilesSummary(0)
	}
}
