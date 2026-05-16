package checkpoint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ApplyRewind restores workspace files from a checkpoint record.
func ApplyRewind(workspace string, rec Record) error {
	for _, f := range rec.Files {
		abs := filepath.Join(workspace, filepath.Clean(f.RelPath))
		ws, err := filepath.Abs(workspace)
		if err != nil {
			return err
		}
		abs, err = filepath.Abs(abs)
		if err != nil {
			return err
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
