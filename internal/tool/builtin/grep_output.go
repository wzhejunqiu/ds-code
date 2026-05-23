package builtin

import "fmt"

// Grep output_mode values (shared with grep tool and TUI display).
const (
	GrepOutputContent           = "content"
	GrepOutputFilesWithMatches  = "files_with_matches"
	GrepOutputCount             = "count"
)

// ParseGrepOutputMode normalizes and validates grep output_mode (empty → files_with_matches).
func ParseGrepOutputMode(s string) (string, error) {
	switch s {
	case "", GrepOutputFilesWithMatches:
		return GrepOutputFilesWithMatches, nil
	case GrepOutputContent, GrepOutputCount:
		return s, nil
	default:
		return "", fmt.Errorf("invalid output_mode: %q", s)
	}
}
