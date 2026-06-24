package rgutil

import (
	"path/filepath"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/permission"
)

// IsGitOnlyPath reports whether path is .git or under .git/.
func IsGitOnlyPath(path string) bool {
	path = filepath.ToSlash(strings.Trim(path, "/"))
	return path == ".git" || strings.HasPrefix(path, ".git/")
}

// RelPathFromWorkspace normalizes raw to a slash-separated path relative to perm.Workspace.
func RelPathFromWorkspace(perm *permission.Engine, raw string) (string, bool) {
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
	rel = filepath.ToSlash(rel)
	rel = strings.TrimPrefix(rel, "./")
	return rel, true
}
