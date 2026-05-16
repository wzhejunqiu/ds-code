package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// WriteFileTool creates or overwrites a whole file.
type WriteFileTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	Strict bool
}

func (t *WriteFileTool) Name() string { return "write_file" }

func (t *WriteFileTool) Description() string {
	return "Create a new file or overwrite an entire file. Use apply_patch to edit existing files."
}

func (t *WriteFileTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"path": map[string]any{
			"type":        "string",
			"description": "Relative path from project root",
		},
		"content": map[string]any{
			"type":        "string",
			"description": "Full file contents",
		},
	}, []string{"path", "content"}, t.Strict)
}

func (t *WriteFileTool) PermissionLevel() permission.Level { return permission.LevelHigh }

func (t *WriteFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if in.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	abs, err := t.Perm.ResolvePath(in.Path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, []byte(in.Content), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("wrote %s (%d bytes)", in.Path, len(in.Content)), nil
}

var _ tool.Tool = (*WriteFileTool)(nil)
