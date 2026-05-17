package apply

import (
	"fmt"
	"github.com/hejunqiu/ds-code/internal/patch"
	"os"
	"path/filepath"
	"strings"
)

// Options configures patch application.
type Options struct {
	MaxChangedLines int
}

type fileBackup struct {
	data []byte
	mode os.FileMode
}

// Apply parses and applies a patch under workspace with backup/rollback on failure.
// resolve must return absolute paths; each result is verified to lie under workspace.
func Apply(workspace string, patchText string, resolve func(rel string) (string, error), opts Options) (summary string, err error) {
	changes, err := patch.Parse(patchText)
	if err != nil {
		return "", err
	}
	if opts.MaxChangedLines > 0 {
		n, err := patch.CountChangedLines(patchText)
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
		case patch.ChangeAdd:
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

		case patch.ChangeDelete:
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

		case patch.ChangeUpdate:
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
