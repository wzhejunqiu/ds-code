package glob

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
)

// GlobTool finds paths matching a glob under the workspace.
type GlobTool struct {
	Cfg       *config.Config
	Perm      *permission.Engine
	Gitignore *tool.GitignoreMatcher
	Strict    bool
}

func (t *GlobTool) Name() string { return "glob" }

func (t *GlobTool) Description() string { return DescGlob }

func (t *GlobTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"pattern": map[string]any{"type": "string", "description": builtin.SchemaGlobPattern},
		"path":    map[string]any{"type": "string", "description": builtin.SchemaPathRelDefault},
	}, []string{"pattern"}, t.Strict)
}

func (t *GlobTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *GlobTool) gitignoreIgnored(rel string) bool {
	return t.Gitignore != nil && t.Gitignore.Ignored(rel)
}

func (t *GlobTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var in struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if in.Pattern == "" {
		return "", fmt.Errorf("%s", builtin.ErrPatternRequired)
	}
	base := in.Path
	if base == "" {
		base = "."
	}
	root, err := t.Perm.CheckReadablePath(base)
	if err != nil {
		return "", err
	}
	limit := t.Cfg.Tools.Glob.MaxResults
	if limit <= 0 {
		limit = builtin.DefaultMaxResults
	}

	candidates, err := builtin.CollectGlobPattern(ctx, t.Perm, root, in.Pattern, builtin.FileFilter{}, t.gitignoreIgnored)
	if err != nil {
		return "", err
	}
	builtin.SortByModTimeDesc(candidates,
		func(c builtin.FileCandidate) time.Time { return c.ModTime },
		func(c builtin.FileCandidate) string { return c.Rel },
	)

	var lines []string
	truncated := false
	for _, c := range candidates {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		lines = append(lines, c.Rel)
		if len(lines) >= limit {
			truncated = len(candidates) > limit
			lines = lines[:limit]
			break
		}
	}
	if len(lines) == 0 {
		return ResultNoFilesMatched, nil
	}
	out := strings.Join(lines, "\n")
	if truncated {
		out += "\n" + fmt.Sprintf(builtin.TruncatedAtResults, limit)
	}
	return out, nil
}

var _ tool.Tool = (*GlobTool)(nil)
