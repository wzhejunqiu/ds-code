package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// GrepTool searches file contents with a regex.
type GrepTool struct {
	Cfg       *config.Config
	Perm      *permission.Engine
	Gitignore *tool.GitignoreMatcher
	Strict    bool
}

func (t *GrepTool) Name() string { return "grep" }

func (t *GrepTool) Description() string {
	return "Search for a regex pattern in files under the workspace. Respects .gitignore."
}

func (t *GrepTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"pattern": map[string]any{"type": "string", "description": "Regular expression"},
		"path":    map[string]any{"type": "string", "description": "Directory or file relative to project root (default .)"},
	}, []string{"pattern"}, t.Strict)
}

func (t *GrepTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *GrepTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
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
		return "", fmt.Errorf("pattern is required")
	}
	if len(in.Pattern) > 512 {
		return "", fmt.Errorf("pattern too long (max 512)")
	}
	re, err := regexp.Compile(in.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex: %w", err)
	}
	searchPath := in.Path
	if searchPath == "" {
		searchPath = "."
	}
	root, err := t.Perm.CheckReadablePath(searchPath)
	if err != nil {
		return "", err
	}

	limit := t.Cfg.Tools.Grep.HeadLimit
	var matches []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			if permission.IsSensitiveAbs(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if permission.IsSensitiveAbs(path) {
			return nil
		}
		relWalk, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel := filepath.ToSlash(relWalk)
		if searchPath != "." {
			rel = filepath.ToSlash(filepath.Join(searchPath, relWalk))
		}
		if t.Gitignore != nil && t.Gitignore.Ignored(rel) {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > 2*1024*1024 {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lines := strings.Split(string(b), "\n")
		for i, line := range lines {
			if len(line) > 64*1024 {
				continue
			}
			if re.MatchString(line) {
				matches = append(matches, fmt.Sprintf("%s:%d:%s", filepath.ToSlash(rel), i+1, strings.TrimSpace(line)))
				if len(matches) >= limit {
					return errStopWalk
				}
			}
		}
		return nil
	})
	if err != nil && err != errStopWalk {
		return "", err
	}
	if len(matches) == 0 {
		return "no matches", nil
	}
	out := strings.Join(matches, "\n")
	if len(matches) >= limit {
		out += fmt.Sprintf("\n... truncated at %d matches", limit)
	}
	return out, nil
}

var errStopWalk = fmt.Errorf("stop")

var _ tool.Tool = (*GrepTool)(nil)
