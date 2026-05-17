package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/lsp"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// DiagnosticsTool runs LSP diagnostics on workspace paths.
type DiagnosticsTool struct {
	Cfg       *config.Config
	Perm      *permission.Engine
	Gitignore *tool.GitignoreMatcher
	LSP       *lsp.Manager
	Strict    bool
}

func (t *DiagnosticsTool) Name() string { return "diagnostics" }

func (t *DiagnosticsTool) Description() string {
	return "Run language-server diagnostics on files or directories (gopls, tsserver, clangd, etc.)."
}

func (t *DiagnosticsTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"paths": map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string"},
			"description": "Files or directories relative to project root",
		},
		"severity": map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string"},
			"description": "Filter: error, warning, info, hint",
		},
	}, []string{"paths"}, t.Strict)
}

func (t *DiagnosticsTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *DiagnosticsTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if t.LSP == nil || !t.Cfg.LSP.Enabled {
		return "", fmt.Errorf("LSP is disabled in config")
	}
	var in struct {
		Paths    []string `json:"paths"`
		Severity []string `json:"severity"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if len(in.Paths) == 0 {
		return "", fmt.Errorf("paths is required")
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
		return strings.Join(notes, "\n") + "\nNo diagnostic files found.", nil
	}

	byServer := make(map[string][]string)
	reg := t.LSP.Registry()
	for _, rel := range files {
		sid := lsp.ServerForExt(reg, lsp.NormalizeExt(rel))
		if sid == "" {
			notes = append(notes, fmt.Sprintf("--- skip %s: no LSP server for extension", rel))
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
				notes = append(notes, fmt.Sprintf("--- %s: %v", rel, err))
				continue
			}
			content, err := os.ReadFile(abs)
			if err != nil {
				notes = append(notes, fmt.Sprintf("--- %s: %v", rel, err))
				continue
			}
			diags, err := client.OpenFile(ctx, rel, content, sevFilter, maxIssues)
			if err != nil {
				notes = append(notes, fmt.Sprintf("--- %s: %v", rel, err))
				continue
			}
			for _, d := range diags {
				lines = append(lines, fmt.Sprintf("%s:%d:%d [%s] %s", rel, d.Line, d.Col, d.Severity, d.Message))
			}
			if len(diags) == 0 {
				lines = append(lines, fmt.Sprintf("%s: (no issues)", rel))
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
		return "No diagnostics.", nil
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
		abs, err := t.Perm.CheckReadablePath(p)
		if err != nil {
			notes = append(notes, fmt.Sprintf("--- %s: %v", p, err))
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			notes = append(notes, fmt.Sprintf("--- %s: %v", p, err))
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
				if permission.IsSensitiveAbs(path) {
					return filepath.SkipDir
				}
				return nil
			}
			if permission.IsSensitiveAbs(path) {
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
			if t.Gitignore != nil && t.Gitignore.Ignored(rel) {
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
