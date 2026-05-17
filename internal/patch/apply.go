package patch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ApplyOptions configures patch application.
type ApplyOptions struct {
	MaxChangedLines int
}

// Apply parses and applies a patch under workspace with backup/rollback on failure.
func Apply(workspace string, patchText string, resolve func(rel string) (string, error), opts ApplyOptions) (summary string, err error) {
	changes, err := Parse(patchText)
	if err != nil {
		return "", err
	}
	if opts.MaxChangedLines > 0 {
		n, err := CountChangedLines(patchText)
		if err != nil {
			return "", err
		}
		if n > opts.MaxChangedLines {
			return "", fmt.Errorf("patch changes %d lines, limit %d", n, opts.MaxChangedLines)
		}
	}

	backups := make(map[string][]byte)
	written := make([]string, 0)

	rollback := func() {
		for _, p := range written {
			if b, ok := backups[p]; ok {
				if b == nil {
					_ = os.Remove(p)
				} else {
					_ = os.WriteFile(p, b, 0o644)
				}
			}
		}
	}

	defer func() {
		if err != nil {
			rollback()
		}
	}()

	var applied []string
	for _, ch := range changes {
		switch ch.Kind {
		case ChangeAdd:
			abs, err := resolve(ch.Path)
			if err != nil {
				return "", err
			}
			if _, err := os.Stat(abs); err == nil {
				return "", fmt.Errorf("add file exists: %s", ch.Path)
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return "", err
			}
			content := strings.Join(ch.AddLines, "\n")
			if len(ch.AddLines) > 0 {
				content += "\n"
			}
			backups[abs] = nil
			if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
				return "", err
			}
			written = append(written, abs)
			applied = append(applied, "add "+ch.Path)

		case ChangeDelete:
			abs, err := resolve(ch.Path)
			if err != nil {
				return "", err
			}
			b, err := os.ReadFile(abs)
			if err != nil {
				return "", fmt.Errorf("delete %s: %w", ch.Path, err)
			}
			backups[abs] = b
			if err := os.Remove(abs); err != nil {
				return "", err
			}
			written = append(written, abs)
			applied = append(applied, "delete "+ch.Path)

		case ChangeUpdate:
			abs, err := resolve(ch.Path)
			if err != nil {
				return "", err
			}
			orig, err := os.ReadFile(abs)
			if err != nil {
				return "", fmt.Errorf("update %s: %w", ch.Path, err)
			}
			backups[abs] = append([]byte(nil), orig...)
			lines := splitLines(string(orig))
			for _, chunk := range ch.Chunks {
				lines, err = applyChunk(lines, chunk)
				if err != nil {
					return "", fmt.Errorf("%s: %w", ch.Path, err)
				}
			}
			newContent := joinLines(lines)
			dest := abs
			if ch.MoveTo != "" {
				dest, err = resolve(ch.MoveTo)
				if err != nil {
					return "", err
				}
				if _, err := os.Stat(dest); err == nil {
					return "", fmt.Errorf("move target exists: %s", ch.MoveTo)
				}
				if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
					return "", err
				}
				if dest != abs {
					if b, err := os.ReadFile(dest); err == nil {
						backups[dest] = b
					} else {
						backups[dest] = nil
					}
				}
			}
			if err := os.WriteFile(dest, []byte(newContent), 0o644); err != nil {
				return "", err
			}
			written = append(written, dest)
			if ch.MoveTo != "" && dest != abs {
				_ = os.Remove(abs)
				applied = append(applied, fmt.Sprintf("update %s -> %s", ch.Path, ch.MoveTo))
			} else {
				applied = append(applied, "update "+ch.Path)
			}
		default:
			return "", fmt.Errorf("unknown change kind %q", ch.Kind)
		}
	}
	return strings.Join(applied, "; "), nil
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func applyChunk(lines []string, chunk Chunk) ([]string, error) {
	if len(chunk.Old) == 0 && len(chunk.New) > 0 {
		// pure insert at end or after context
		if chunk.Context != "" {
			idx := findContext(lines, chunk.Context)
			if idx < 0 {
				return nil, fmt.Errorf("context not found: %q", chunk.Context)
			}
			out := append(append([]string{}, lines[:idx+1]...), chunk.New...)
			return append(out, lines[idx+1:]...), nil
		}
		return append(lines, chunk.New...), nil
	}
	idx := findSubslice(lines, chunk.Old)
	if idx < 0 && chunk.Context != "" {
		ctx := findContext(lines, chunk.Context)
		if ctx >= 0 {
			idx = findSubslice(lines[ctx:], chunk.Old)
			if idx >= 0 {
				idx += ctx
			}
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("hunk not found (%d old lines)", len(chunk.Old))
	}
	out := append(append([]string{}, lines[:idx]...), chunk.New...)
	out = append(out, lines[idx+len(chunk.Old):]...)
	if chunk.EOF && idx+len(chunk.New) < len(out) {
		// no-op validation: EOF marker accepted
	}
	return out, nil
}

func findContext(lines []string, ctx string) int {
	for i, l := range lines {
		if strings.Contains(l, ctx) {
			return i
		}
	}
	return -1
}

func findSubslice(haystack, needle []string) int {
	if len(needle) == 0 {
		return len(haystack)
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
