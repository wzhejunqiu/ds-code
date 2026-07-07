package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/patch"
)

// FilePreview holds before/after content for one patched file (read-only preview).
type FilePreview struct {
	Path     string
	Original string
	Modified string
}

// Preview computes post-patch file contents without writing to disk.
func Preview(workspace string, patchText string, resolve func(rel string) (string, error)) ([]FilePreview, error) {
	validate := func(rel string) error {
		_, err := resolve(rel)
		return err
	}
	changes, err := patch.Parse(patchText, validate)
	if err != nil {
		return nil, err
	}
	out := make([]FilePreview, 0, len(changes))
	for _, ch := range changes {
		switch ch.Kind {
		case patch.ChangeAdd:
			content := joinLines(ch.AddLines)
			if !strings.HasSuffix(content, "\n") && len(ch.AddLines) > 0 {
				content += "\n"
			}
			out = append(out, FilePreview{Path: ch.Path, Original: "", Modified: content})

		case patch.ChangeDelete:
			abs, err := resolve(ch.Path)
			if err != nil {
				return nil, err
			}
			b, err := os.ReadFile(abs)
			if err != nil {
				return nil, fmt.Errorf("delete %s: %w", ch.Path, err)
			}
			out = append(out, FilePreview{Path: ch.Path, Original: string(b), Modified: ""})

		case patch.ChangeUpdate:
			abs, err := resolve(ch.Path)
			if err != nil {
				return nil, err
			}
			b, err := os.ReadFile(abs)
			if err != nil {
				return nil, fmt.Errorf("update %s: %w", ch.Path, err)
			}
			lines := splitLines(string(b))
			for _, chunk := range ch.Chunks {
				lines, err = applyChunk(lines, chunk)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", ch.Path, err)
				}
			}
			modified := joinLines(lines)
			displayPath := ch.Path
			if ch.MoveTo != "" {
				displayPath = ch.Path + " → " + ch.MoveTo
			}
			out = append(out, FilePreview{Path: displayPath, Original: string(b), Modified: modified})

		default:
			return nil, fmt.Errorf("unknown change kind %q", ch.Kind)
		}
	}
	return out, nil
}

// ResolveWorkspacePath returns an absolute path under workspace for preview.
func ResolveWorkspacePath(workspace, rel string) (string, error) {
	abs := filepath.Join(workspace, filepath.FromSlash(rel))
	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	ws, err := filepath.Abs(workspace)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, ws+string(filepath.Separator)) && abs != ws {
		return "", fmt.Errorf("path outside workspace: %s", rel)
	}
	return abs, nil
}
