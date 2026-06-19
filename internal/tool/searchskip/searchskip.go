package searchskip

import (
	"path/filepath"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
)

// Matcher applies .git hard skip and optional user-configured skip_dirs during Agent enumeration.
type Matcher struct {
	skipDirs []string
}

// New creates a Matcher from relative workspace-root path segments.
func New(skipDirs []string) *Matcher {
	var cleaned []string
	for _, d := range skipDirs {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		d = filepath.ToSlash(strings.Trim(d, "/"))
		if strings.Contains(d, "..") || filepath.IsAbs(d) {
			logging.L().Warn("searchskip: ignoring invalid skip_dirs entry", zap.String("entry", d))
			continue
		}
		cleaned = append(cleaned, d)
	}
	return &Matcher{skipDirs: cleaned}
}

// Ignored reports whether rel (slash path from workspace root) should be filtered from results.
func (m *Matcher) Ignored(rel string) bool {
	return m.IgnoredInScope(rel, ".")
}

// IgnoredInScope reports whether rel should be filtered when enumeration scope is scopeRoot
// (the tool path parameter). .git is always filtered; skip_dirs are bypassed when scopeRoot
// explicitly targets that directory or a subdirectory.
func (m *Matcher) IgnoredInScope(rel, scopeRoot string) bool {
	if m == nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".git" || strings.HasPrefix(rel, ".git/") {
		return true
	}
	scopeRoot = filepath.ToSlash(strings.Trim(scopeRoot, "/"))
	if scopeRoot == "" {
		scopeRoot = "."
	}
	if scopeRoot != "." {
		for _, skip := range m.skipDirs {
			if scopeRoot == skip || strings.HasPrefix(scopeRoot, skip+"/") {
				return false
			}
		}
	}
	return m.matchesSkipDir(rel)
}

// SkipDir reports whether a directory rel should be skipped during Walk (includes .git).
func (m *Matcher) SkipDir(rel string) bool {
	rel = filepath.ToSlash(strings.Trim(rel, "/"))
	if rel == "" {
		return false
	}
	if rel == ".git" || strings.HasPrefix(rel, ".git/") {
		return true
	}
	return m.matchesSkipDir(rel)
}

func (m *Matcher) matchesSkipDir(rel string) bool {
	for _, skip := range m.skipDirs {
		if rel == skip || strings.HasPrefix(rel, skip+"/") {
			return true
		}
	}
	return false
}

// ShouldSkipWalkDir returns whether Walk should SkipDir for relDir.
// Explicit path into a skip dir (relDir equals skip entry) is still entered.
func (m *Matcher) ShouldSkipWalkDir(relDir, walkRoot string) bool {
	relDir = filepath.ToSlash(strings.Trim(relDir, "/"))
	walkRoot = filepath.ToSlash(strings.Trim(walkRoot, "/"))
	if relDir == ".git" || strings.HasPrefix(relDir, ".git/") {
		return true
	}
	// When user explicitly sets path to a skip dir, allow entry.
	if walkRoot != "" && walkRoot != "." {
		for _, skip := range m.skipDirs {
			if walkRoot == skip || strings.HasPrefix(walkRoot, skip+"/") {
				return false
			}
		}
	}
	return m.matchesSkipDir(relDir)
}
