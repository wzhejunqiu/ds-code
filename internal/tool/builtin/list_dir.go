package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// ListDirTool lists directory entries under the workspace.
type ListDirTool struct {
	Cfg       *config.Config
	Perm      *permission.Engine
	Gitignore *tool.GitignoreMatcher
	Strict    bool
}

func (t *ListDirTool) Name() string { return "list_dir" }

func (t *ListDirTool) Description() string {
	return "List files and directories at a path relative to project root."
}

func (t *ListDirTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"path": map[string]any{"type": "string", "description": "Directory path (default .)"},
	}, nil, t.Strict)
}

func (t *ListDirTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *ListDirTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var in struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(args, &in)
	if in.Path == "" {
		in.Path = "."
	}
	root, err := t.Perm.CheckReadablePath(in.Path)
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	limit := t.Cfg.Tools.Glob.MaxResults
	if limit <= 0 {
		limit = 200
	}
	var lines []string
	for _, e := range entries {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		name := e.Name()
		if name == ".git" {
			continue
		}
		rel := filepath.Join(in.Path, name)
		if t.Gitignore != nil && t.Gitignore.Ignored(rel) {
			continue
		}
		abs := filepath.Join(root, name)
		if permission.IsSensitiveAbs(abs) {
			continue
		}
		if e.IsDir() {
			lines = append(lines, name+"/")
		} else {
			lines = append(lines, name)
		}
		if len(lines) >= limit {
			lines = append(lines, fmt.Sprintf("... truncated at %d entries", limit))
			break
		}
	}
	if len(lines) == 0 {
		return "(empty)", nil
	}
	return strings.Join(lines, "\n"), nil
}
