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

type fileBackup struct {
	data []byte
	mode os.FileMode
}

// Apply parses and applies a patch under workspace with backup/rollback on failure.
// resolve must return absolute paths; each result is verified to lie under workspace.
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

	backups := make(map[string]fileBackup)
	written := make([]string, 0)

	rollback := func() {
		for _, p := range written {
			b, ok := backups[p]
			if !ok || b.data == nil {
				_ = os.Remove(p)
				continue
			}
			_ = os.WriteFile(p, b.data, b.mode)
		}
	}

	defer func() {
		if err != nil {
			rollback()
		}
	}()

	resolveChecked := func(rel string) (string, error) {
		abs, err := resolve(rel)
		if err != nil {
			return "", err
		}
		if err := ensureUnderWorkspace(workspace, abs); err != nil {
			return "", err
		}
		return abs, nil
	}

	var applied []string
	for _, ch := range changes {
		switch ch.Kind {
		case ChangeAdd:
			abs, err := resolveChecked(ch.Path)
			if err != nil {
				return "", err
			}
			if _, err := os.Stat(abs); err == nil {
				return "", fmt.Errorf("add file exists: %s", ch.Path)
			} else if err != nil && !os.IsNotExist(err) {
				return "", fmt.Errorf("add %s: %w", ch.Path, err)
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return "", err
			}
			content := strings.Join(ch.AddLines, "\n")
			if len(ch.AddLines) > 0 {
				content += "\n"
			}
			backups[abs] = fileBackup{data: nil, mode: 0o644}
			if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
				return "", err
			}
			written = append(written, abs)
			applied = append(applied, "add "+ch.Path)

		case ChangeDelete:
			abs, err := resolveChecked(ch.Path)
			if err != nil {
				return "", err
			}
			b, mode, err := readFileBackup(abs)
			if err != nil {
				return "", fmt.Errorf("delete %s: %w", ch.Path, err)
			}
			backups[abs] = fileBackup{data: b, mode: mode}
			if err := os.Remove(abs); err != nil {
				return "", err
			}
			written = append(written, abs)
			applied = append(applied, "delete "+ch.Path)

		case ChangeUpdate:
			abs, err := resolveChecked(ch.Path)
			if err != nil {
				return "", err
			}
			orig, mode, err := readFileBackup(abs)
			if err != nil {
				return "", fmt.Errorf("update %s: %w", ch.Path, err)
			}
			backups[abs] = fileBackup{data: append([]byte(nil), orig...), mode: mode}
			lines := splitLines(string(orig))
			for _, chunk := range ch.Chunks {
				lines, err = applyChunk(lines, chunk)
				if err != nil {
					return "", fmt.Errorf("%s: %w", ch.Path, err)
				}
			}
			newContent := joinLines(lines)
			dest := abs
			destMode := mode
			if ch.MoveTo != "" {
				dest, err = resolveChecked(ch.MoveTo)
				if err != nil {
					return "", err
				}
				if _, err := os.Stat(dest); err == nil {
					return "", fmt.Errorf("move target exists: %s", ch.MoveTo)
				} else if err != nil && !os.IsNotExist(err) {
					return "", fmt.Errorf("move %s: %w", ch.MoveTo, err)
				}
				if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
					return "", err
				}
				if dest != abs {
					if b, m, err := readFileBackup(dest); err == nil {
						backups[dest] = fileBackup{data: b, mode: m}
					} else if os.IsNotExist(err) {
						backups[dest] = fileBackup{data: nil, mode: destMode}
					} else {
						return "", fmt.Errorf("move %s: %w", ch.MoveTo, err)
					}
				}
			}
			if err := os.WriteFile(dest, []byte(newContent), destMode); err != nil {
				return "", err
			}
			written = append(written, dest)
			if ch.MoveTo != "" && dest != abs {
				if err := os.Remove(abs); err != nil {
					return "", fmt.Errorf("move remove %s: %w", ch.Path, err)
				}
				written = append(written, abs)
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

func ensureUnderWorkspace(workspace, abs string) error {
	ws, err := filepath.Abs(workspace)
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(ws); err == nil {
		ws = resolved
	}
	target, err := filepath.Abs(abs)
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(target); err == nil {
		target = resolved
	} else if parent, err := filepath.EvalSymlinks(filepath.Dir(target)); err == nil {
		target = filepath.Join(parent, filepath.Base(target))
	}
	rel, err := filepath.Rel(ws, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("path outside workspace: %s", abs)
	}
	return nil
}

func readFileBackup(abs string) ([]byte, os.FileMode, error) {
	st, err := os.Stat(abs)
	if err != nil {
		return nil, 0, err
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return nil, 0, err
	}
	return b, st.Mode().Perm(), nil
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
		if chunk.Context != "" {
			ctx, err := findContext(lines, chunk.Context)
			if err != nil {
				return nil, err
			}
			if ctx < 0 {
				return nil, fmt.Errorf("context not found: %q", chunk.Context)
			}
			out := append(append([]string{}, lines[:ctx+1]...), chunk.New...)
			return append(out, lines[ctx+1:]...), nil
		}
		return append(lines, chunk.New...), nil
	}

	var idx int
	var err error
	if chunk.Context != "" {
		ctx, ctxErr := findContext(lines, chunk.Context)
		if ctxErr != nil {
			return nil, ctxErr
		}
		if ctx < 0 {
			return nil, fmt.Errorf("context not found: %q", chunk.Context)
		}
		searchFrom := ctx
		idx, err = findSubsliceUnique(lines[searchFrom:], chunk.Old)
		if err != nil {
			return nil, err
		}
		if idx >= 0 {
			idx += searchFrom
		}
	} else {
		idx, err = findSubsliceUnique(lines, chunk.Old)
		if err != nil {
			return nil, err
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("hunk not found (%d old lines)", len(chunk.Old))
	}
	if chunk.EOF && idx+len(chunk.Old) != len(lines) {
		return nil, fmt.Errorf("EOF hunk must end at end of file")
	}
	out := append(append([]string{}, lines[:idx]...), chunk.New...)
	out = append(out, lines[idx+len(chunk.Old):]...)
	return out, nil
}

func findContext(lines []string, ctx string) (int, error) {
	ctx = strings.TrimSpace(ctx)
	if ctx == "" {
		return -1, nil
	}
	var exact []int
	for i, l := range lines {
		if strings.TrimSpace(l) == ctx {
			exact = append(exact, i)
		}
	}
	switch len(exact) {
	case 1:
		return exact[0], nil
	case 0:
		// fall through to unique substring match
	default:
		return -1, fmt.Errorf("ambiguous context %q: %d matches", ctx, len(exact))
	}
	var substr []int
	for i, l := range lines {
		if strings.Contains(l, ctx) {
			substr = append(substr, i)
		}
	}
	switch len(substr) {
	case 0:
		return -1, nil
	case 1:
		return substr[0], nil
	default:
		return -1, fmt.Errorf("ambiguous context %q: %d matches", ctx, len(substr))
	}
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

func findSubsliceUnique(haystack, needle []string) (int, error) {
	idx := findSubslice(haystack, needle)
	if idx < 0 {
		return -1, nil
	}
	if findSubslice(haystack[idx+len(needle):], needle) >= 0 {
		return -1, fmt.Errorf("ambiguous hunk: multiple matches")
	}
	return idx, nil
}
