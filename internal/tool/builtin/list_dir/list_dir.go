package list_dir

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

// ListDirTool lists directory entries under the workspace.
type ListDirTool struct {
	Cfg        *config.Config
	Perm       *permission.Engine
	SearchSkip *searchskip.Matcher
	Strict     bool
}

func (t *ListDirTool) Name() string { return tool.NameListDir.String() }

func (t *ListDirTool) WithPerm(perm *permission.Engine) tool.Tool {
	cp := *t
	cp.Perm = perm
	return &cp
}

func (t *ListDirTool) IsReadOnly() bool        { return true }
func (t *ListDirTool) IsConcurrencySafe() bool { return true }

func (t *ListDirTool) Description() string { return DescListDir }

func (t *ListDirTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"path": map[string]any{"type": "string", "description": builtin.SchemaPathDirDefault},
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
	scopePath := filepath.ToSlash(strings.Trim(in.Path, "/"))
	if scopePath == ".git" {
		return ResultEmpty, nil
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
		limit = builtin.DefaultMaxResults
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
		if t.SearchSkip != nil && t.SearchSkip.IgnoredInScope(rel, in.Path) {
			continue
		}
		abs := filepath.Join(root, name)
		if t.Perm.SkipSensitiveAbs(abs) {
			continue
		}
		if e.IsDir() {
			lines = append(lines, name+"/")
		} else {
			lines = append(lines, name)
		}
		if len(lines) >= limit {
			lines = append(lines, fmt.Sprintf(builtin.TruncatedAtEntries, limit))
			break
		}
	}
	if len(lines) == 0 {
		return ResultEmpty, nil
	}
	return strings.Join(lines, "\n"), nil
}

var _ tool.Tool = (*ListDirTool)(nil)
