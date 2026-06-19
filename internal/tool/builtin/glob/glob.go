package glob

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

// GlobTool finds paths matching a glob under the workspace.
type GlobTool struct {
	Cfg        *config.Config
	Perm       *permission.Engine
	SearchSkip *searchskip.Matcher
	Strict     bool
}

func (t *GlobTool) Name() string { return tool.NameGlob.String() }

func (t *GlobTool) WithPerm(perm *permission.Engine) tool.Tool {
	cp := *t
	cp.Perm = perm
	return &cp
}

func (t *GlobTool) IsReadOnly() bool        { return true }
func (t *GlobTool) IsConcurrencySafe() bool { return true }

func (t *GlobTool) Description() string { return DescGlob }

func (t *GlobTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"pattern": map[string]any{"type": "string", "description": builtin.SchemaGlobPattern},
		"path":    map[string]any{"type": "string", "description": builtin.SchemaPathRelDefault},
	}, []string{"pattern"}, t.Strict)
}

func (t *GlobTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *GlobTool) searchIgnored(rel, scopeRoot string) bool {
	if t.SearchSkip == nil {
		return false
	}
	return t.SearchSkip.IgnoredInScope(rel, scopeRoot)
}

func (t *GlobTool) searchSkipWalk(relFromRoot, base string) bool {
	if t.SearchSkip == nil {
		fullRel := filepath.ToSlash(relFromRoot)
		if base != "" && base != "." {
			fullRel = filepath.ToSlash(filepath.Join(base, relFromRoot))
		}
		return fullRel == ".git" || strings.HasPrefix(fullRel, ".git/")
	}
	fullRel := filepath.ToSlash(relFromRoot)
	if base != "" && base != "." {
		fullRel = filepath.ToSlash(filepath.Join(base, relFromRoot))
	}
	return t.SearchSkip.ShouldSkipWalkDir(fullRel, base)
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

	candidates, err := builtin.CollectGlobPattern(ctx, t.Perm, root, in.Pattern, builtin.FileFilter{}, func(rel string) bool {
		return t.searchIgnored(rel, base)
	}, func(relFromRoot string) bool {
		return t.searchSkipWalk(relFromRoot, base)
	})
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
