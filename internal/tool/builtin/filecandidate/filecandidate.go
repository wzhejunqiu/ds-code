package filecandidate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/globmatch"
	"github.com/wzhejunqiu/ds-code/internal/tool/textfile"
)

// FileCandidate is a searchable file under the workspace.
type FileCandidate struct {
	AbsPath string
	Rel     string // relative to project root, slash-separated
	ModTime time.Time
}

// FileFilter controls MakeFileCandidate filtering.
type FileFilter struct {
	MaxFileBytes int64 // 0 = no size limit
}

// ValidateGlobMatches ensures glob results stay inside the workspace.
func ValidateGlobMatches(perm *permission.Engine, matches []string, pattern string) error {
	for _, m := range matches {
		abs := filepath.Clean(m)
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		}
		if err := perm.CheckAbsPath(abs, permission.PathBoundary); err != nil {
			return permission.GlobOutsideWorkspaceError(abs, pattern)
		}
	}
	return nil
}

// MakeFileCandidate returns a candidate when absPath passes filters.
// Rel is always relative to perm.Workspace (project root).
func MakeFileCandidate(perm *permission.Engine, absPath string, filter FileFilter) *FileCandidate {
	if perm.SkipSensitiveAbs(absPath) {
		return nil
	}
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return nil
	}
	if filter.MaxFileBytes > 0 && info.Size() > filter.MaxFileBytes {
		return nil
	}
	if !textfile.IsSearchable(absPath) {
		return nil
	}
	ws, err := filepath.Abs(perm.Workspace)
	if err != nil {
		return nil
	}
	if resolved, err := filepath.EvalSymlinks(ws); err == nil {
		ws = resolved
	}
	abs := absPath
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		abs = resolved
	}
	rel, err := filepath.Rel(ws, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil
	}
	return &FileCandidate{
		AbsPath: absPath,
		Rel:     filepath.ToSlash(rel),
		ModTime: info.ModTime(),
	}
}

// CollectGlobPattern matches pattern under root and returns filtered file candidates.
func CollectGlobPattern(
	ctx context.Context,
	perm *permission.Engine,
	root, pattern string,
	filter FileFilter,
	ignored func(rel string) bool,
	skipDir func(rel string) bool,
) ([]FileCandidate, error) {
	skipSensitive := func(abs string) bool { return perm.SkipSensitiveAbs(abs) }
	absPaths, err := globmatch.MatchFiles(root, pattern, 0, skipDir, skipSensitive)
	if err != nil {
		return nil, err
	}
	if err := ValidateGlobMatches(perm, absPaths, pattern); err != nil {
		return nil, err
	}

	var out []FileCandidate
	for _, abs := range absPaths {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		c := MakeFileCandidate(perm, abs, filter)
		if c == nil {
			continue
		}
		if ignored != nil && ignored(c.Rel) {
			continue
		}
		out = append(out, *c)
	}
	return out, nil
}
