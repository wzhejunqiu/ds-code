package grep

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/rgutil"
	"github.com/wzhejunqiu/ds-code/internal/tool/textfile"
)

// ErrRipgrepTimeout is returned when the ripgrep subprocess exceeds tools.grep.timeout.
var ErrRipgrepTimeout = rgutil.ErrTimeout

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
	if rgutil.IsGitOnlyPath(in.Path) {
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

	rgPath, err := rgutil.ResolveBinary(t.Cfg.Tools.Grep)
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
	stdout, _, err := rgutil.Exec(ctx, rgPath, args, t.Perm.Workspace, timeout)
	if err != nil {
		if errors.Is(err, rgutil.ErrTimeout) {
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

func buildRipgrepArgs(t *GrepTool, in grepInput, mode string) ([]string, error) {
	args := []string{
		"--json",
		"--max-filesize", "2M",
	}
	args = append(args, rgutil.IgnoreFlags(t.Cfg.Tools.Grep.RespectGitignore, false)...)
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
	for _, g := range rgutil.SensitiveExcludeGlobs() {
		args = append(args, "--glob", g)
	}
	for _, g := range rgutil.SkipGlobs(in.Path, t.Cfg.Tools.Search.SkipDirs) {
		args = append(args, "--glob", g)
	}
	if !rgutil.IsGitOnlyPath(in.Path) {
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
			rel, ok := rgutil.RelPathFromWorkspace(perm, msg.Data.Path.Text)
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
			rel, ok := rgutil.RelPathFromWorkspace(perm, msg.Data.Path.Text)
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
	rgutil.SortFilesByMtime(files, perm)
	meta := grepPageMeta{
		Limit:        limit,
		Offset:       offset,
		TotalEntries: len(files),
		TotalFiles:   len(files),
	}
	files = rgutil.PaginateStrings(files, offset, limit)
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
	lines = rgutil.PaginateStrings(lines, offset, limit)
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
			order = append(order, fileOrder{rel: r.Rel, modTime: rgutil.FileModTime(perm, r.Rel), first: i})
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
	rgutil.SortFilesByMtime(files, perm)
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
	countLines = rgutil.PaginateStrings(countLines, offset, limit)
	return formatGrepOutput(builtin.GrepOutputCount, countLines, meta), nil
}

// relPathFromWorkspace is a test helper alias for rgutil.RelPathFromWorkspace.
func relPathFromWorkspace(perm *permission.Engine, raw string) (string, bool) {
	return rgutil.RelPathFromWorkspace(perm, raw)
}

// resolveRipgrepBinary is a test helper alias for rgutil.ResolveBinary.
func resolveRipgrepBinary(cfg config.GrepToolConfig) (string, error) {
	return rgutil.ResolveBinary(cfg)
}
