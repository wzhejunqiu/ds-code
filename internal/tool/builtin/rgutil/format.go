package rgutil

import (
	"fmt"
	"strings"
)

// FilesPageMeta holds pagination metadata for file-list output.
type FilesPageMeta struct {
	Limit        int
	Offset       int
	TotalEntries int
	TotalFiles   int
}

// FormatPaginationFooter returns the standard pagination footer line.
func FormatPaginationFooter(limit, offset int) string {
	return fmt.Sprintf("[Showing results with pagination = limit: %d, offset: %d]", limit, offset)
}

// FormatFilesSummary returns the "Found N files" summary line.
func FormatFilesSummary(totalFiles int) string {
	return fmt.Sprintf("Found %d files", totalFiles)
}

func needsPaginationFooter(meta FilesPageMeta, shown int) bool {
	if meta.Offset > 0 {
		return true
	}
	if meta.Limit > 0 && meta.TotalEntries > meta.Offset+shown {
		return true
	}
	return false
}

// FormatFilesOutput formats file paths with summary and optional pagination footer.
func FormatFilesOutput(paths []string, meta FilesPageMeta) string {
	if meta.TotalFiles == 0 {
		return FormatFilesSummary(0)
	}
	var b strings.Builder
	b.WriteString(FormatFilesSummary(meta.TotalFiles))
	for _, p := range paths {
		b.WriteByte('\n')
		b.WriteString(p)
	}
	if needsPaginationFooter(meta, len(paths)) {
		b.WriteByte('\n')
		b.WriteString(FormatPaginationFooter(meta.Limit, meta.Offset))
	}
	return b.String()
}

// PaginateStrings applies offset and limit to a string slice.
func PaginateStrings(items []string, offset, limit int) []string {
	if offset > 0 {
		if offset >= len(items) {
			return nil
		}
		items = items[offset:]
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}
