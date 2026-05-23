package checkpoint

import (
	"fmt"
	"os"

	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/patch"
	"go.uber.org/zap"
)

const maxCaptureBytes = 4 << 20 // 4 MiB per file

// CapturePaths reads current file bytes for relative paths under workspace.
func CapturePaths(workspace string, resolve func(rel string) (string, error), paths []string) ([]FileState, error) {
	seen := make(map[string]struct{})
	var out []FileState
	for _, rel := range paths {
		if rel == "" {
			continue
		}
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		st, err := captureOne(workspace, resolve, rel)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	var totalBytes int
	for _, st := range out {
		totalBytes += len(st.Content)
	}
	logging.L().Debug("checkpoint capture",
		zap.Int("paths", len(out)),
		zap.Int("bytes", totalBytes),
	)
	return out, nil
}

func captureOne(workspace string, resolve func(rel string) (string, error), rel string) (FileState, error) {
	abs, err := resolve(rel)
	if err != nil {
		return FileState{}, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return FileState{RelPath: rel, Existed: false}, nil
		}
		return FileState{}, err
	}
	if len(data) > maxCaptureBytes {
		return FileState{}, fmt.Errorf("checkpoint: %s exceeds %d byte capture limit", rel, maxCaptureBytes)
	}
	return FileState{RelPath: rel, Existed: true, Content: data}, nil
}

// PathsFromTool extracts affected relative paths for checkpoint capture.
func PathsFromTool(tool string, workspace string, args map[string]any) ([]string, error) {
	switch tool {
	case "write_file":
		if p, _ := args["path"].(string); p != "" {
			return []string{p}, nil
		}
	case "apply_patch":
		if patchText, _ := args["patch"].(string); patchText != "" {
			return patch.Paths(patchText, workspace)
		}
	}
	return nil, nil
}
