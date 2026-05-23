package glob

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
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
	"github.com/hejunqiu/ds-code/internal/tool/globmatch"
	"github.com/hejunqiu/ds-code/internal/tool/textfile"
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
		limit = 200
	}

	matchLimit := 0
	if strings.Contains(in.Pattern, "**") {
		matchLimit = limit * 4
	}
	matches, err := globmatch.MatchFiles(root, in.Pattern, matchLimit)
	if err != nil {
		return "", err
	}
	if err := t.validateGlobMatches(matches, in.Pattern); err != nil {
		return "", err
	}

	var lines []string
	for _, m := range matches {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if permission.IsSensitiveAbs(m) {
			continue
		}
		rel, err := filepath.Rel(root, m)
		if err != nil {
			continue
		}
		if t.Gitignore != nil && t.Gitignore.Ignored(rel) {
			continue
		}
		info, err := os.Stat(m)
		if err != nil || info.IsDir() {
			continue
		}
		if !textfile.IsSearchable(m) {
			continue
		}
		lines = append(lines, rel)
		if len(lines) >= limit {
			lines = append(lines, fmt.Sprintf(builtin.TruncatedAtResults, limit))
			break
		}
	}
	if len(lines) == 0 {
		return ResultNoFilesMatched, nil
	}
	return strings.Join(lines, "\n"), nil
}

func (t *GlobTool) validateGlobMatches(matches []string, pattern string) error {
	for _, m := range matches {
		abs := filepath.Clean(m)
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		}
		if err := t.Perm.EnsureAbsUnderWorkspace(abs, pattern); err != nil {
			return permission.GlobOutsideWorkspaceError(abs, pattern)
		}
	}
	return nil
}

var _ tool.Tool = (*GlobTool)(nil)
