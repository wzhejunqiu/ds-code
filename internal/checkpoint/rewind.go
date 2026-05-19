package checkpoint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hejunqiu/ds-code/internal/patch"
)

// ApplyRewind restores workspace files from a checkpoint record.
func ApplyRewind(workspace string, rec Record) error {
	ws, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		ws, err = filepath.Abs(workspace)
		if err != nil {
			return err
		}
	}
	for _, f := range rec.Files {
		if err := patch.ValidatePath(workspace, f.RelPath); err != nil {
			return err
		}
		abs := filepath.Join(ws, filepath.Clean(f.RelPath))
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		}
		rel, err := filepath.Rel(ws, abs)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("checkpoint: path outside workspace: %s", f.RelPath)
		}
		if f.Existed {
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(abs, f.Content, 0o644); err != nil {
				return err
			}
			continue
		}
		if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
