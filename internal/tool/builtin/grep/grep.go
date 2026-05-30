package grep

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/globmatch"
)

const (
	maxFileBytes = 2 * 1024 * 1024
	maxLineBytes = 64 * 1024
	maxWorkers   = 8
)

type outputMode string

const (
	modeContent          outputMode = builtin.GrepOutputContent
	modeFilesWithMatches outputMode = builtin.GrepOutputFilesWithMatches
	modeCount            outputMode = builtin.GrepOutputCount
)

// GrepTool searches file contents with a regex.
type GrepTool struct {
	Cfg       *config.Config
	Perm      *permission.Engine
	Gitignore *tool.GitignoreMatcher
	Strict    bool
}

func (t *GrepTool) Name() string { return tool.NameGrep.String() }

func (t *GrepTool) WithPerm(perm *permission.Engine) tool.Tool {
	cp := *t
	cp.Perm = perm
	return &cp
}

func (t *GrepTool) IsReadOnly() bool        { return true }
func (t *GrepTool) IsConcurrencySafe() bool { return true }

func (t *GrepTool) Description() string { return DescGrep }

func (t *GrepTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"pattern": map[string]any{"type": "string", "description": SchemaRegexPattern},
		"path":    map[string]any{"type": "string", "description": SchemaGrepPath},
		"output_mode": map[string]any{
			"type": "string",
			"enum": []string{
				builtin.GrepOutputContent,
				builtin.GrepOutputFilesWithMatches,
				builtin.GrepOutputCount,
			},
			"description": SchemaOutputMode,
		},
	}, []string{"pattern"}, t.Strict)
}

func (t *GrepTool) PermissionLevel() permission.Level { return permission.LevelLow }

type lineHit struct {
	lineNum int
	text    string
}

type fileHits struct {
	rel  string
	hits []lineHit
}

type searchResult struct {
	lines      []string
	totalCount int
	truncated  bool
}

func (t *GrepTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var in struct {
		Pattern    string `json:"pattern"`
		Path       string `json:"path"`
		OutputMode string `json:"output_mode"`
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
	modeStr, err := builtin.ParseGrepOutputMode(in.OutputMode)
	if err != nil {
		return "", err
	}
	mode := outputMode(modeStr)

	re, err := regexp.Compile(in.Pattern)
	if err != nil {
		return "", fmt.Errorf("%s: %w", builtin.ErrInvalidRegex, err)
	}
	searchPath := in.Path
	if searchPath == "" {
		searchPath = "."
	}

	limit := t.Cfg.Tools.Grep.HeadLimit
	if limit <= 0 {
		limit = 200
	}
	searchLimit := limit
	if mode == modeCount {
		searchLimit = 0
	}

	candidates, err := t.collectCandidates(ctx, searchPath)
	if err != nil {
		return "", err
	}
	res := t.searchCandidates(ctx, candidates, re, mode, searchLimit)
	if err := ctx.Err(); err != nil {
		return "", err
	}

	switch mode {
	case modeCount:
		return strconv.Itoa(res.totalCount), nil
	default:
		if len(res.lines) == 0 {
			return builtin.ResultGrepNoMatches, nil
		}
		out := strings.Join(res.lines, "\n")
		if res.truncated {
			truncFmt := builtin.TruncatedAtMatches
			if mode == modeFilesWithMatches {
				truncFmt = builtin.TruncatedAtPaths
			}
			out += fmt.Sprintf("\n"+truncFmt, limit)
		}
		return out, nil
	}
}

func (t *GrepTool) gitignoreIgnored(rel string) bool {
	return t.Gitignore != nil && t.Gitignore.Ignored(rel)
}

func (t *GrepTool) collectCandidates(ctx context.Context, searchPath string) ([]builtin.FileCandidate, error) {
	if globmatch.HasMeta(searchPath) {
		return t.collectGlobPath(ctx, searchPath)
	}
	return t.collectExactPath(ctx, searchPath)
}

func (t *GrepTool) collectExactPath(ctx context.Context, searchPath string) ([]builtin.FileCandidate, error) {
	root, err := t.Perm.CheckReadablePath(searchPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	filter := builtin.FileFilter{MaxFileBytes: maxFileBytes}
	if !info.IsDir() {
		if c := builtin.MakeFileCandidate(t.Perm, root, filter); c != nil {
			return []builtin.FileCandidate{*c}, nil
		}
		return nil, nil
	}

	var out []builtin.FileCandidate
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
		if c := builtin.MakeFileCandidate(t.Perm, path, filter); c != nil {
			if t.Gitignore != nil && t.Gitignore.Ignored(c.Rel) {
				return nil
			}
			out = append(out, *c)
		}
		return nil
	})
	return out, err
}

func (t *GrepTool) collectGlobPath(ctx context.Context, searchPath string) ([]builtin.FileCandidate, error) {
	base, pattern := globmatch.SplitPath(searchPath)
	root, err := t.Perm.CheckReadablePath(base)
	if err != nil {
		return nil, err
	}
	return builtin.CollectGlobPattern(ctx, t.Perm, root, pattern, builtin.FileFilter{MaxFileBytes: maxFileBytes}, t.gitignoreIgnored)
}

func (t *GrepTool) searchCandidates(ctx context.Context, candidates []builtin.FileCandidate, re *regexp.Regexp, mode outputMode, limit int) searchResult {
	builtin.SortByModTimeDesc(candidates,
		func(c builtin.FileCandidate) time.Time { return c.ModTime },
		func(c builtin.FileCandidate) string { return c.Rel },
	)

	workers := maxWorkers
	if n := runtime.NumCPU(); n < workers {
		workers = n
	}
	if workers < 1 {
		workers = 1
	}

	var res searchResult
	stopAfter := 0
	if mode == modeFilesWithMatches {
		stopAfter = 1
	}

	for i := 0; i < len(candidates); {
		if err := ctx.Err(); err != nil {
			break
		}
		if mode != modeCount && limit > 0 {
			switch mode {
			case modeContent:
				if len(res.lines) >= limit {
					res.truncated = true
					return res
				}
			case modeFilesWithMatches:
				if len(res.lines) >= limit {
					res.truncated = true
					return res
				}
			}
		}

		end := i + workers
		if end > len(candidates) {
			end = len(candidates)
		}
		batch := candidates[i:end]
		batchHits := make([]fileHits, len(batch))
		var wg sync.WaitGroup
		for j, c := range batch {
			wg.Add(1)
			go func(j int, c builtin.FileCandidate) {
				defer wg.Done()
				if ctx.Err() != nil {
					return
				}
				batchHits[j] = grepFile(c.AbsPath, c.Rel, re, stopAfter)
			}(j, c)
		}
		wg.Wait()

		for _, fh := range batchHits {
			if len(fh.hits) == 0 {
				continue
			}
			switch mode {
			case modeCount:
				res.totalCount += len(fh.hits)
			case modeFilesWithMatches:
				res.lines = append(res.lines, fh.rel)
				if limit > 0 && len(res.lines) >= limit {
					res.truncated = end < len(candidates)
					return res
				}
			case modeContent:
				for hi, h := range fh.hits {
					res.lines = append(res.lines, fmt.Sprintf("%s:%d:%s", fh.rel, h.lineNum, h.text))
					if limit > 0 && len(res.lines) >= limit {
						res.truncated = end < len(candidates) || hi < len(fh.hits)-1
						return res
					}
				}
			}
		}
		i = end
	}
	return res
}

func grepFile(absPath, rel string, re *regexp.Regexp, stopAfter int) fileHits {
	b, err := os.ReadFile(absPath)
	if err != nil {
		return fileHits{rel: rel}
	}
	var hits []lineHit
	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		if len(line) > maxLineBytes {
			continue
		}
		if re.MatchString(line) {
			hits = append(hits, lineHit{
				lineNum: i + 1,
				text:    strings.TrimSpace(line),
			})
			if stopAfter > 0 && len(hits) >= stopAfter {
				break
			}
		}
	}
	return fileHits{rel: rel, hits: hits}
}

var _ tool.Tool = (*GrepTool)(nil)
