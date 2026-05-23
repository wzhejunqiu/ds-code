package grep

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
	"github.com/hejunqiu/ds-code/internal/tool/globmatch"
	"github.com/hejunqiu/ds-code/internal/tool/textfile"
)

const (
	maxFileBytes = 2 * 1024 * 1024
	maxLineBytes = 64 * 1024
	maxWorkers   = 8
)

// GrepTool searches file contents with a regex.
type GrepTool struct {
	Cfg       *config.Config
	Perm      *permission.Engine
	Gitignore *tool.GitignoreMatcher
	Strict    bool
}

func (t *GrepTool) Name() string { return "grep" }

func (t *GrepTool) Description() string { return DescGrep }

func (t *GrepTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"pattern": map[string]any{"type": "string", "description": SchemaRegexPattern},
		"path":    map[string]any{"type": "string", "description": SchemaGrepPath},
	}, []string{"pattern"}, t.Strict)
}

func (t *GrepTool) PermissionLevel() permission.Level { return permission.LevelLow }

type fileCandidate struct {
	absPath string
	rel     string
	modTime time.Time
}

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
		return "", fmt.Errorf("%s", builtin.ErrPatternRequired)
	}
	if len(in.Pattern) > 512 {
		return "", fmt.Errorf("%s", builtin.ErrPatternTooLong)
	}
	re, err := regexp.Compile(in.Pattern)
	if err != nil {
		return "", fmt.Errorf("%s: %w", builtin.ErrInvalidRegex, err)
	}
	searchPath := in.Path
	if searchPath == "" {
		searchPath = "."
	}

	limit := t.Cfg.Tools.Grep.HeadLimit
	candidates, err := t.collectCandidates(ctx, searchPath)
	if err != nil {
		return "", err
	}
	matches := t.searchCandidates(ctx, candidates, re, limit)
	if len(matches) == 0 {
		return builtin.ResultGrepNoMatches, nil
	}
	out := strings.Join(matches, "\n")
	if len(matches) >= limit {
		out += fmt.Sprintf("\n"+builtin.TruncatedAtMatches, limit)
	}
	return out, nil
}

func (t *GrepTool) collectCandidates(ctx context.Context, searchPath string) ([]fileCandidate, error) {
	if globmatch.HasMeta(searchPath) {
		return t.collectGlobPath(ctx, searchPath)
	}
	return t.collectExactPath(ctx, searchPath)
}

func (t *GrepTool) collectExactPath(ctx context.Context, searchPath string) ([]fileCandidate, error) {
	root, err := t.Perm.CheckReadablePath(searchPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if c := t.makeCandidate(root); c != nil {
			return []fileCandidate{*c}, nil
		}
		return nil, nil
	}

	var out []fileCandidate
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
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
		if c := t.makeCandidate(path); c != nil {
			if t.Gitignore != nil && t.Gitignore.Ignored(c.rel) {
				return nil
			}
			out = append(out, *c)
		}
		return nil
	})
	return out, err
}

func (t *GrepTool) collectGlobPath(ctx context.Context, searchPath string) ([]fileCandidate, error) {
	base, pattern := globmatch.SplitPath(searchPath)
	root, err := t.Perm.CheckReadablePath(base)
	if err != nil {
		return nil, err
	}
	absPaths, err := globmatch.MatchFiles(root, pattern, 0)
	if err != nil {
		return nil, err
	}
	if err := t.validateGlobMatches(absPaths, searchPath); err != nil {
		return nil, err
	}

	var out []fileCandidate
	for _, abs := range absPaths {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		if c := t.makeCandidate(abs); c != nil {
			if t.Gitignore != nil && t.Gitignore.Ignored(c.rel) {
				continue
			}
			out = append(out, *c)
		}
	}
	return out, nil
}

func (t *GrepTool) makeCandidate(absPath string) *fileCandidate {
	if permission.IsSensitiveAbs(absPath) {
		return nil
	}
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() || info.Size() > maxFileBytes {
		return nil
	}
	if !textfile.IsSearchable(absPath) {
		return nil
	}
	rel, err := filepath.Rel(t.Perm.Workspace, absPath)
	if err != nil {
		return nil
	}
	return &fileCandidate{
		absPath: absPath,
		rel:     filepath.ToSlash(rel),
		modTime: info.ModTime(),
	}
}

func (t *GrepTool) validateGlobMatches(matches []string, pattern string) error {
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

func (t *GrepTool) searchCandidates(ctx context.Context, candidates []fileCandidate, re *regexp.Regexp, limit int) []string {
	sort.Slice(candidates, func(i, j int) bool {
		if !candidates[i].modTime.Equal(candidates[j].modTime) {
			return candidates[i].modTime.After(candidates[j].modTime)
		}
		return candidates[i].rel < candidates[j].rel
	})

	workers := maxWorkers
	if n := runtime.NumCPU(); n < workers {
		workers = n
	}
	if workers < 1 {
		workers = 1
	}

	var matches []string
	for i := 0; i < len(candidates) && len(matches) < limit; {
		if err := ctx.Err(); err != nil {
			break
		}
		end := i + workers
		if end > len(candidates) {
			end = len(candidates)
		}
		batch := candidates[i:end]
		batchMatches := make([][]string, len(batch))
		var wg sync.WaitGroup
		for j, c := range batch {
			wg.Add(1)
			go func(j int, c fileCandidate) {
				defer wg.Done()
				if ctx.Err() != nil {
					return
				}
				batchMatches[j] = grepFile(c.absPath, c.rel, re)
			}(j, c)
		}
		wg.Wait()
		for _, lines := range batchMatches {
			for _, line := range lines {
				matches = append(matches, line)
				if len(matches) >= limit {
					return matches
				}
			}
		}
		i = end
	}
	return matches
}

func grepFile(absPath, rel string, re *regexp.Regexp) []string {
	b, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}
	var matches []string
	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		if len(line) > maxLineBytes {
			continue
		}
		if re.MatchString(line) {
			matches = append(matches, fmt.Sprintf("%s:%d:%s", rel, i+1, strings.TrimSpace(line)))
		}
	}
	return matches
}

var _ tool.Tool = (*GrepTool)(nil)
