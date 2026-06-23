package grep

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/grep/rgbin"
	"github.com/wzhejunqiu/ds-code/internal/tool/textfile"
)

// ErrRipgrepTimeout is returned when the ripgrep subprocess exceeds tools.grep.timeout.
var ErrRipgrepTimeout = errors.New("ripgrep: search timed out")

type recordKind int

const (
	recordMatch recordKind = iota
	recordContext
)

type grepRecord struct {
	Kind recordKind
	Rel  string
	Line int
	Text string
}

func runRipgrep(ctx context.Context, t *GrepTool, in grepInput) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	mode, err := builtin.ParseGrepOutputMode(in.OutputMode)
	if err != nil {
		return "", err
	}
	if isGitOnlyPath(in.Path) {
		return emptyOutputForMode(mode), nil
	}

	searchPath := in.Path
	if searchPath == "" {
		searchPath = "."
	}
	if _, err := t.Perm.CheckReadablePath(searchPath); err != nil {
		return "", err
	}

	limit := resolveHeadLimit(in, t.Cfg)
	offset := in.Offset
	if offset < 0 {
		return "", fmt.Errorf("offset must be non-negative")
	}

	rgPath, err := resolveRipgrepBinary(t.Cfg.Tools.Grep)
	if err != nil {
		return "", err
	}
	args, err := buildRipgrepArgs(t, in, mode)
	if err != nil {
		return "", err
	}

	timeout := t.Cfg.Tools.Grep.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	stdout, _, err := execRipgrep(ctx, rgPath, args, t.Perm.Workspace, timeout)
	if err != nil {
		if errors.Is(err, ErrRipgrepTimeout) {
			return "", fmt.Errorf("%s", MsgRipgrepTimeout)
		}
		return "", err
	}

	records, err := parseRipgrepJSON(stdout, t.Perm)
	if err != nil {
		return "", err
	}
	showLineNums := true
	if in.LineNums != nil {
		showLineNums = *in.LineNums
	}
	return postProcess(ctx, records, mode, limit, offset, showLineNums, t.Perm)
}

func isGitOnlyPath(path string) bool {
	path = filepath.ToSlash(strings.Trim(path, "/"))
	return path == ".git" || strings.HasPrefix(path, ".git/")
}

func resolveHeadLimit(in grepInput, cfg *config.Config) int {
	if in.HeadLimit != nil {
		return *in.HeadLimit
	}
	limit := cfg.Tools.Grep.HeadLimit
	if limit <= 0 {
		limit = 250
	}
	return limit
}

func resolveRipgrepBinary(cfg config.GrepToolConfig) (string, error) {
	switch cfg.Binary {
	case "", "bundled":
		return rgbin.Path()
	case "system":
		path, err := exec.LookPath("rg")
		if err != nil {
			return "", fmt.Errorf("ripgrep: rg not found in PATH")
		}
		return path, nil
	case "path":
		if cfg.BinaryPath == "" {
			return "", fmt.Errorf("tools.grep.binary_path required when binary=path")
		}
		return validateExecutable(cfg.BinaryPath)
	default:
		return "", fmt.Errorf("unknown tools.grep.binary: %q", cfg.Binary)
	}
}

func validateExecutable(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("ripgrep: binary_path: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("ripgrep: binary_path is a directory")
	}
	if info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("ripgrep: binary_path is not executable")
	}
	return path, nil
}

func buildRipgrepArgs(t *GrepTool, in grepInput, mode string) ([]string, error) {
	args := []string{
		"--json",
		"--max-filesize", "2M",
	}
	if !t.Cfg.Tools.Grep.RespectGitignore {
		args = append(args, "--no-ignore", "--no-ignore-vcs", "--no-ignore-global")
	}
	if in.IgnoreCase {
		args = append(args, "--ignore-case")
	}
	if in.Multiline {
		args = append(args, "-U", "--multiline-dotall")
	}
	if in.Type != "" {
		args = append(args, "--type", in.Type)
	}
	if in.Glob != "" {
		args = append(args, "--glob", in.Glob)
	}
	for _, g := range sensitiveExcludeGlobs() {
		args = append(args, "--glob", g)
	}
	for _, g := range t.ripgrepSkipGlobs(in.Path) {
		args = append(args, "--glob", g)
	}
	if !isGitOnlyPath(in.Path) {
		args = append(args, "--glob", "!.git/**")
	}

	if mode == builtin.GrepOutputContent {
		showLineNums := true
		if in.LineNums != nil {
			showLineNums = *in.LineNums
		}
		if showLineNums {
			args = append(args, "--line-number")
		}
		ctxLines := in.Context
		if ctxLines == 0 {
			ctxLines = in.ContextC
		}
		if ctxLines > 0 {
			args = append(args, "-C", fmt.Sprintf("%d", ctxLines))
		} else {
			if in.Before > 0 {
				args = append(args, "-B", fmt.Sprintf("%d", in.Before))
			}
			if in.After > 0 {
				args = append(args, "-A", fmt.Sprintf("%d", in.After))
			}
		}
	}

	searchPath := in.Path
	if searchPath == "" {
		searchPath = "."
	}
	args = append(args, "--", in.Pattern, searchPath)
	return args, nil
}

func sensitiveExcludeGlobs() []string {
	return []string{
		"!**/.env", "!**/.env.*", "!**/.envrc",
		"!**/.netrc", "!**/.npmrc", "!**/.pypirc",
		"!**/id_rsa*", "!**/id_ed25519*", "!**/id_ecdsa*", "!**/id_dsa*",
		"!**/*.pem", "!**/*.key",
		"!**/.ssh/**", "!**/.aws/**", "!**/.docker/**", "!**/.kube/**",
		"!**/.gnupg/**", "!**/credentials/**", "!**/secrets/**",
		"!**/token.json", "!**/secrets.json", "!**/credentials.json",
		"!**/service-account.json", "!**/kubeconfig",
	}
}

func (t *GrepTool) ripgrepSkipGlobs(scopeRoot string) []string {
	scope := filepath.ToSlash(strings.Trim(scopeRoot, "/"))
	if scope == "" {
		scope = "."
	}
	var globs []string
	for _, skip := range t.Cfg.Tools.Search.SkipDirs {
		skip = filepath.ToSlash(strings.Trim(skip, "/"))
		if skip == "" {
			continue
		}
		if scope != "." && (scope == skip || strings.HasPrefix(scope, skip+"/")) {
			continue
		}
		globs = append(globs, "!"+skip+"/**")
	}
	return globs
}

func execRipgrep(ctx context.Context, rgPath string, args []string, workspace string, timeout time.Duration) (stdout, stderr string, err error) {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, rgPath, args...)
	cmd.Dir = workspace
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	if runCtx.Err() == context.DeadlineExceeded {
		return outBuf.String(), errBuf.String(), ErrRipgrepTimeout
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return "", "", ctx.Err()
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			if exitErr.ExitCode() == 1 {
				return outBuf.String(), errBuf.String(), nil
			}
			return outBuf.String(), errBuf.String(), mapRipgrepStderr(errBuf.String(), exitErr.ExitCode())
		}
		return outBuf.String(), errBuf.String(), runErr
	}
	return outBuf.String(), errBuf.String(), nil
}

func mapRipgrepStderr(stderr string, code int) error {
	stderr = strings.TrimSpace(stderr)
	if strings.Contains(stderr, "regex parse error") || strings.Contains(stderr, "error parsing regex") {
		if stderr != "" {
			return fmt.Errorf("%s: %s", builtin.ErrInvalidRegex, stderr)
		}
		return fmt.Errorf("%s", builtin.ErrInvalidRegex)
	}
	if stderr != "" {
		return fmt.Errorf("ripgrep exited with code %d: %s", code, stderr)
	}
	return fmt.Errorf("ripgrep exited with code %d", code)
}

type rgJSONLine struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		Lines struct {
			Text string `json:"text"`
		} `json:"lines"`
		LineNumber *int `json:"line_number"`
		Submatches []struct {
			Match struct {
				Text string `json:"text"`
			} `json:"match"`
		} `json:"submatches"`
	} `json:"data"`
}

func parseRipgrepJSON(stdout string, perm *permission.Engine) ([]grepRecord, error) {
	var records []grepRecord
	sc := bufio.NewScanner(strings.NewReader(stdout))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var msg rgJSONLine
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "match":
			rel, ok := relPathFromWorkspace(perm, msg.Data.Path.Text)
			if !ok {
				continue
			}
			abs, err := perm.ResolvePath(rel)
			if err != nil || perm.SkipSensitiveAbs(abs) {
				continue
			}
			if !textfile.IsSearchable(abs) {
				continue
			}
			lineNum := 0
			if msg.Data.LineNumber != nil {
				lineNum = *msg.Data.LineNumber
			}
			text := msg.Data.Lines.Text
			if text == "" && len(msg.Data.Submatches) > 0 {
				text = msg.Data.Submatches[0].Match.Text
			}
			records = append(records, grepRecord{
				Kind: recordMatch,
				Rel:  rel,
				Line: lineNum,
				Text: text,
			})
		case "context":
			rel, ok := relPathFromWorkspace(perm, msg.Data.Path.Text)
			if !ok {
				continue
			}
			abs, err := perm.ResolvePath(rel)
			if err != nil || perm.SkipSensitiveAbs(abs) {
				continue
			}
			if !textfile.IsSearchable(abs) {
				continue
			}
			lineNum := 0
			if msg.Data.LineNumber != nil {
				lineNum = *msg.Data.LineNumber
			}
			records = append(records, grepRecord{
				Kind: recordContext,
				Line: lineNum,
				Rel:  rel,
				Text: msg.Data.Lines.Text,
			})
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func relPathFromWorkspace(perm *permission.Engine, raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	abs := raw
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(perm.Workspace, abs)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	ws := perm.Workspace
	if resolved, err := filepath.EvalSymlinks(ws); err == nil {
		ws = resolved
	}
	rel, err := filepath.Rel(ws, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func postProcess(ctx context.Context, records []grepRecord, mode string, limit, offset int, showLineNums bool, perm *permission.Engine) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	switch mode {
	case builtin.GrepOutputContent:
		return postProcessContent(records, limit, offset, showLineNums, perm)
	case builtin.GrepOutputCount:
		return postProcessCount(records, limit, offset, perm)
	default:
		return postProcessFiles(records, limit, offset, perm)
	}
}

func fileModTime(perm *permission.Engine, rel string) time.Time {
	abs := filepath.Join(perm.Workspace, filepath.FromSlash(rel))
	info, err := os.Stat(abs)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func sortFilesByMtime(files []string, perm *permission.Engine) {
	sort.Slice(files, func(i, j int) bool {
		ti := fileModTime(perm, files[i])
		tj := fileModTime(perm, files[j])
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return files[i] < files[j]
	})
}

func postProcessFiles(records []grepRecord, limit, offset int, perm *permission.Engine) (string, error) {
	seen := make(map[string]bool)
	var files []string
	for _, r := range records {
		if r.Kind != recordMatch {
			continue
		}
		if seen[r.Rel] {
			continue
		}
		seen[r.Rel] = true
		files = append(files, r.Rel)
	}
	sortFilesByMtime(files, perm)
	meta := grepPageMeta{
		Limit:        limit,
		Offset:       offset,
		TotalEntries: len(files),
		TotalFiles:   len(files),
	}
	files = paginateStrings(files, offset, limit)
	return formatGrepOutput(builtin.GrepOutputFilesWithMatches, files, meta), nil
}

func postProcessContent(records []grepRecord, limit, offset int, showLineNums bool, perm *permission.Engine) (string, error) {
	records = sortContentRecords(records, perm)
	var lines []string
	for _, r := range records {
		lines = append(lines, formatGrepLine(r.Kind, r.Rel, r.Line, r.Text, showLineNums))
	}
	meta := grepPageMeta{
		Limit:        limit,
		Offset:       offset,
		TotalEntries: len(lines),
	}
	lines = paginateStrings(lines, offset, limit)
	return formatGrepOutput(builtin.GrepOutputContent, lines, meta), nil
}

func sortContentRecords(records []grepRecord, perm *permission.Engine) []grepRecord {
	type fileOrder struct {
		rel     string
		modTime time.Time
		first   int
	}
	fileIndex := make(map[string]int)
	var order []fileOrder
	for i, r := range records {
		if _, ok := fileIndex[r.Rel]; !ok {
			fileIndex[r.Rel] = len(order)
			order = append(order, fileOrder{rel: r.Rel, modTime: fileModTime(perm, r.Rel), first: i})
		}
	}
	sort.Slice(order, func(i, j int) bool {
		if !order[i].modTime.Equal(order[j].modTime) {
			return order[i].modTime.After(order[j].modTime)
		}
		return order[i].rel < order[j].rel
	})
	rank := make(map[string]int, len(order))
	for i, fo := range order {
		rank[fo.rel] = i
	}
	out := append([]grepRecord(nil), records...)
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := rank[out[i].Rel], rank[out[j].Rel]
		if ri != rj {
			return ri < rj
		}
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func postProcessCount(records []grepRecord, limit, offset int, perm *permission.Engine) (string, error) {
	counts := make(map[string]int)
	totalMatches := 0
	for _, r := range records {
		if r.Kind != recordMatch {
			continue
		}
		counts[r.Rel]++
		totalMatches++
	}
	var files []string
	for rel := range counts {
		files = append(files, rel)
	}
	sortFilesByMtime(files, perm)
	var countLines []string
	for _, rel := range files {
		countLines = append(countLines, fmt.Sprintf("%s:%d", rel, counts[rel]))
	}
	meta := grepPageMeta{
		Limit:        limit,
		Offset:       offset,
		TotalEntries: len(countLines),
		TotalFiles:   len(files),
		TotalMatches: totalMatches,
	}
	countLines = paginateStrings(countLines, offset, limit)
	return formatGrepOutput(builtin.GrepOutputCount, countLines, meta), nil
}

func paginateStrings(items []string, offset, limit int) []string {
	if offset > 0 {
		if offset >= len(items) {
			return nil
		}
		items = items[offset:]
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}
