package glob

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/rgutil"
	"github.com/wzhejunqiu/ds-code/internal/tool/textfile"
)

type globInput struct {
	Pattern      string `json:"pattern"`
	Path         string `json:"path"`
	explicitPath bool
}

func parseGlobInput(args json.RawMessage) (globInput, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil {
		return globInput{}, err
	}
	var in globInput
	if err := json.Unmarshal(args, &in); err != nil {
		return globInput{}, err
	}
	_, in.explicitPath = raw["path"]
	return in, nil
}

func runRipgrepFiles(ctx context.Context, t *GlobTool, in globInput) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	searchPath := in.Path
	if searchPath == "" {
		searchPath = "."
	}

	if in.explicitPath {
		if err := validateExplicitPath(in.Path, t.Perm); err != nil {
			return "", err
		}
	}

	if rgutil.IsGitOnlyPath(searchPath) {
		return rgutil.FormatFilesSummary(0), nil
	}

	limit := t.Cfg.Tools.Glob.MaxResults
	if limit <= 0 {
		limit = builtin.DefaultMaxResults
	}

	rgPath, err := rgutil.ResolveBinary(t.Cfg.Tools.Grep)
	if err != nil {
		return "", err
	}
	args := buildGlobRipgrepArgs(t, in.Pattern, searchPath)

	timeout := t.Cfg.Tools.Grep.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	stdout, _, err := rgutil.Exec(ctx, rgPath, args, t.Perm.Workspace, timeout)
	if err != nil {
		if errors.Is(err, rgutil.ErrTimeout) {
			return "", fmt.Errorf("%s", MsgRipgrepTimeout)
		}
		return "", err
	}

	files, err := parseRipgrepFiles(stdout, t.Perm, t.Cfg.Tools.Glob.IncludeHidden)
	if err != nil {
		return "", err
	}
	return postProcessFiles(ctx, files, limit, t.Perm)
}

func isUNCPath(path string) bool {
	return strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, "//")
}

func validateExplicitPath(inPath string, perm *permission.Engine) error {
	if isUNCPath(inPath) {
		return nil
	}
	ws := perm.Workspace
	if resolved, err := filepath.EvalSymlinks(ws); err == nil {
		ws = resolved
	}
	var abs string
	if filepath.IsAbs(inPath) {
		abs = filepath.Clean(inPath)
	} else {
		abs = filepath.Clean(filepath.Join(ws, inPath))
	}
	rel, err := filepath.Rel(ws, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: path outside workspace: %s", permission.ErrDenied, inPath)
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			cwd, _ := os.Getwd()
			return fmt.Errorf("目录不存在: %s（当前工作目录: %s）", inPath, cwd)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("path 必须是目录，不能是文件: %s", inPath)
	}
	if _, err := perm.CheckReadablePath(inPath); err != nil {
		return err
	}
	return nil
}

func buildGlobRipgrepArgs(t *GlobTool, pattern, searchPath string) []string {
	cfg := t.Cfg.Tools.Glob
	args := []string{"--files"}
	args = append(args, rgutil.IgnoreFlags(cfg.RespectGitignore, cfg.IncludeHidden)...)
	args = append(args, "--glob", pattern)
	for _, g := range rgutil.SensitiveExcludeGlobs() {
		args = append(args, "--glob", g)
	}
	for _, g := range rgutil.SkipGlobs(searchPath, t.Cfg.Tools.Search.SkipDirs) {
		args = append(args, "--glob", g)
	}
	if !rgutil.IsGitOnlyPath(searchPath) {
		args = append(args, "--glob", "!.git/**")
	}
	if searchPath == "" {
		searchPath = "."
	}
	args = append(args, "--", searchPath)
	return args
}

func parseRipgrepFiles(stdout string, perm *permission.Engine, includeHidden bool) ([]string, error) {
	seen := make(map[string]bool)
	var files []string
	sc := bufio.NewScanner(strings.NewReader(stdout))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		rel, ok := rgutil.RelPathFromWorkspace(perm, line)
		if !ok {
			continue
		}
		if !includeHidden && isHiddenRelPath(rel) {
			continue
		}
		if seen[rel] {
			continue
		}
		abs, err := perm.ResolvePath(rel)
		if err != nil || perm.SkipSensitiveAbs(abs) {
			continue
		}
		if !textfile.IsSearchable(abs) {
			continue
		}
		seen[rel] = true
		files = append(files, rel)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

func isHiddenRelPath(rel string) bool {
	for _, seg := range strings.Split(rel, "/") {
		if seg != "" && strings.HasPrefix(seg, ".") {
			return true
		}
	}
	return false
}

func postProcessFiles(ctx context.Context, files []string, limit int, perm *permission.Engine) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	rgutil.SortFilesByMtime(files, perm)
	total := len(files)
	if limit > 0 && total > limit {
		files = files[:limit]
	}
	return formatGlobOutput(files, total, limit), nil
}
