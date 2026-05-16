package context

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
)

var atRefPattern = regexp.MustCompile(`@([a-zA-Z0-9_./\-]+)`)

// AtExpander resolves @file and @dir/ references into user message text.
type AtExpander struct {
	Cfg       *config.Config
	Perm      *permission.Engine
	Gitignore *tool.GitignoreMatcher
}

// Expand parses @refs, enforces budgets, and appends file contents to the message.
func (e *AtExpander) Expand(userText string) (string, error) {
	matches := atRefPattern.FindAllStringSubmatchIndex(userText, -1)
	if len(matches) == 0 {
		return userText, nil
	}

	refs := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, m := range matches {
		ref := userText[m[2]:m[3]]
		if !seen[ref] {
			seen[ref] = true
			refs = append(refs, ref)
		}
	}

	cleaned := atRefPattern.ReplaceAllString(userText, "")
	cleaned = strings.TrimSpace(cleaned)

	maxTotal := e.Cfg.Context.AtReferenceMaxChars
	if maxTotal <= 0 {
		maxTotal = 128000
	}
	remaining := maxTotal
	perFileMax := maxTotal / 10
	if perFileMax < 1 {
		perFileMax = 1
	}

	var blocks []string
	for _, ref := range refs {
		if remaining <= 0 {
			blocks = append(blocks, fmt.Sprintf("--- @%s ---\n[skipped: at_reference budget exhausted]", ref))
			continue
		}
		block, used, err := e.expandRef(ref, perFileMax, remaining)
		if err != nil {
			blocks = append(blocks, fmt.Sprintf("--- @%s ---\nerror: %v", ref, err))
			continue
		}
		remaining -= used
		blocks = append(blocks, block)
	}

	var b strings.Builder
	if cleaned != "" {
		b.WriteString(cleaned)
	}
	for _, block := range blocks {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(block)
	}
	return b.String(), nil
}

func (e *AtExpander) expandRef(ref string, perFileMax, remaining int) (string, int, error) {
	isDir := strings.HasSuffix(ref, "/")
	refPath := strings.TrimSuffix(ref, "/")

	if err := e.Perm.Check("read_file", map[string]any{"path": refPath}); err != nil {
		return "", 0, err
	}
	abs, err := e.Perm.ResolvePath(refPath)
	if err != nil {
		return "", 0, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", 0, err
	}
	if info.IsDir() {
		isDir = true
	}
	if isDir {
		return e.expandDir(ref, abs, perFileMax, remaining)
	}
	return e.expandFile(ref, abs, perFileMax, remaining)
}

func (e *AtExpander) expandFile(ref, abs string, perFileMax, remaining int) (string, int, error) {
	limit := perFileMax
	if limit > remaining {
		limit = remaining
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", 0, err
	}
	content := string(b)
	truncated := false
	if len(content) > limit {
		content = content[:limit]
		truncated = true
	}
	rel, _ := filepath.Rel(e.Perm.Workspace, abs)
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- @%s (%s) ---\n", ref, filepath.ToSlash(rel))
	sb.WriteString(content)
	if truncated {
		sb.WriteString("\n... [file truncated for @ reference budget]")
	}
	return sb.String(), len(content), nil
}

func (e *AtExpander) expandDir(ref, abs string, perFileMax, remaining int) (string, int, error) {
	maxFiles := e.Cfg.Context.AtDirMaxFiles
	if maxFiles <= 0 {
		maxFiles = 50
	}
	maxDepth := e.Cfg.Context.AtDirMaxDepth
	if maxDepth <= 0 {
		maxDepth = 4
	}

	type entry struct {
		rel string
	}
	var files []entry
	err := filepath.WalkDir(abs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if path != abs && d.Name() == ".git" {
				return filepath.SkipDir
			}
			relToRoot, _ := filepath.Rel(abs, path)
			depth := strings.Count(filepath.ToSlash(relToRoot), "/") + 1
			if path != abs && depth > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(e.Perm.Workspace, path)
		if e.Gitignore != nil && e.Gitignore.Ignored(rel) {
			return nil
		}
		files = append(files, entry{rel: filepath.ToSlash(rel)})
		if len(files) >= maxFiles+1 {
			return errStopAtRef
		}
		return nil
	})
	if err != nil && err != errStopAtRef {
		return "", 0, err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "--- @%s/ (directory) ---\n", strings.TrimSuffix(ref, "/"))
	if len(files) > maxFiles {
		b.WriteString(fmt.Sprintf("Too many files (%d+). Listed first %d; use grep/glob for more.\n\n", maxFiles+1, maxFiles))
		files = files[:maxFiles]
	}

	used := 0
	for _, f := range files {
		if remaining-used <= 0 {
			b.WriteString("\n... [remaining files skipped: budget exhausted]")
			break
		}
		full, err := e.Perm.ResolvePath(f.rel)
		if err != nil {
			continue
		}
		block, n, err := e.expandFile(f.rel, full, perFileMax, remaining-used)
		if err != nil {
			fmt.Fprintf(&b, "\n%s: error: %v\n", f.rel, err)
			continue
		}
		b.WriteString("\n")
		b.WriteString(block)
		used += n
	}
	return b.String(), used, nil
}

var errStopAtRef = fmt.Errorf("stop walk")
