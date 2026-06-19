package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/lsp"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

// DiagnosticsTool runs LSP diagnostics on workspace paths.
type DiagnosticsTool struct {
	Cfg        *config.Config
	Perm       *permission.Engine
	SearchSkip *searchskip.Matcher
	LSP        *lsp.Manager
	Strict     bool
}

func (t *DiagnosticsTool) Name() string { return tool.NameDiagnostics.String() }

func (t *DiagnosticsTool) WithPerm(perm *permission.Engine) tool.Tool {
	cp := *t
	cp.Perm = perm
	return &cp
}

func (t *DiagnosticsTool) IsReadOnly() bool        { return true }
func (t *DiagnosticsTool) IsConcurrencySafe() bool { return true }

func (t *DiagnosticsTool) Description() string { return DescDiagnostics }

func (t *DiagnosticsTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"paths": map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string"},
			"description": builtin.SchemaPathsRelRoot,
		},
		"severity": map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string"},
			"description": SchemaSeverity,
		},
	}, []string{"paths"}, t.Strict)
}

func (t *DiagnosticsTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *DiagnosticsTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if t.LSP == nil || !t.Cfg.LSP.Enabled {
		return "", fmt.Errorf("%s", ErrLSPDisabled)
	}
	var in struct {
		Paths    []string `json:"paths"`
		Severity []string `json:"severity"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if len(in.Paths) == 0 {
		return "", fmt.Errorf("%s", builtin.ErrPathsRequired)
	}
	sevFilter := parseSeverityFilter(in.Severity)
	maxFiles := t.Cfg.LSP.MaxFilesPerCall
	if maxFiles <= 0 {
		maxFiles = 10
	}
	maxIssues := t.Cfg.LSP.MaxIssuesPerFile
	if maxIssues <= 0 {
		maxIssues = 20
	}

	files, notes, err := t.collectFiles(ctx, in.Paths, maxFiles)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return strings.Join(notes, "\n") + NoteNoDiagnosticFiles, nil
	}

	byServer := make(map[string][]string)
	reg := t.LSP.Registry()
	for _, rel := range files {
		sid := lsp.ServerForExt(reg, lsp.NormalizeExt(rel))
		if sid == "" {
			notes = append(notes, fmt.Sprintf(NoteSkipNoServer, rel))
			continue
		}
		byServer[sid] = append(byServer[sid], rel)
	}

	var lines []string
	for sid, paths := range byServer {
		client, err := t.LSP.EnsureClient(ctx, sid)
		if err != nil {
			notes = append(notes, "--- "+err.Error())
			continue
		}
		for _, rel := range paths {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			abs, err := t.Perm.CheckReadablePath(rel)
			if err != nil {
				notes = append(notes, fmt.Sprintf(NoteRelPathError, rel, err))
				continue
			}
			content, err := os.ReadFile(abs)
			if err != nil {
				notes = append(notes, fmt.Sprintf(NoteRelPathError, rel, err))
				continue
			}
			diags, err := client.OpenFile(ctx, rel, content, sevFilter, maxIssues)
			if err != nil {
				notes = append(notes, fmt.Sprintf(NoteRelPathError, rel, err))
				continue
			}
			for _, d := range diags {
				lines = append(lines, fmt.Sprintf("%s:%d:%d [%s] %s", rel, d.Line, d.Col, d.Severity, d.Message))
			}
			if len(diags) == 0 {
				lines = append(lines, fmt.Sprintf(ResultNoIssues, rel))
			}
		}
	}
	out := strings.Join(lines, "\n")
	if len(notes) > 0 {
		if out != "" {
			out += "\n"
		}
		out += strings.Join(notes, "\n")
	}
	if out == "" {
		return ResultNoDiagnostics, nil
	}
	return ctxpkg.TruncateToolResult(out, t.Cfg), nil
}

func parseSeverityFilter(in []string) map[string]bool {
	if len(in) == 0 {
		return map[string]bool{"error": true, "warning": true}
	}
	m := make(map[string]bool)
	for _, s := range in {
		m[strings.ToLower(strings.TrimSpace(s))] = true
	}
	return m
}

func (t *DiagnosticsTool) collectFiles(ctx context.Context, paths []string, maxFiles int) ([]string, []string, error) {
	seen := make(map[string]struct{})
	var files []string
	var notes []string
	for _, p := range paths {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		scopePath := filepath.ToSlash(strings.Trim(p, "/"))
		if scopePath == ".git" {
			continue
		}
		abs, err := t.Perm.CheckReadablePath(p)
		if err != nil {
			notes = append(notes, fmt.Sprintf(NotePathError, p, err))
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			notes = append(notes, fmt.Sprintf(NotePathError, p, err))
			continue
		}
		if !info.IsDir() {
			if t.tryAddFile(seen, &files, p, maxFiles) {
				break
			}
			continue
		}
		_ = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if d.Name() == ".git" {
					return filepath.SkipDir
				}
				if t.Perm.SkipSensitiveAbs(path) {
					return filepath.SkipDir
				}
				return nil
			}
			if t.Perm.SkipSensitiveAbs(path) {
				return nil
			}
			relWalk, err := filepath.Rel(abs, path)
			if err != nil {
				return nil
			}
			rel := relWalk
			if p != "." {
				rel = filepath.Join(p, relWalk)
			}
			rel = filepath.ToSlash(rel)
			if t.SearchSkip != nil && t.SearchSkip.IgnoredInScope(rel, p) {
				return nil
			}
			if lsp.ServerForExt(t.LSP.Registry(), lsp.NormalizeExt(rel)) == "" {
				return nil
			}
			if t.tryAddFile(seen, &files, rel, maxFiles) {
				return filepath.SkipAll
			}
			return nil
		})
	}
	return files, notes, nil
}

func (t *DiagnosticsTool) tryAddFile(seen map[string]struct{}, files *[]string, rel string, max int) bool {
	if _, err := t.Perm.CheckReadablePath(rel); err != nil {
		return len(*files) >= max
	}
	if _, ok := seen[rel]; ok {
		return len(*files) >= max
	}
	seen[rel] = struct{}{}
	*files = append(*files, rel)
	return len(*files) >= max
}
