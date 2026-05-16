package tool

import (
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// GitignoreMatcher loads .gitignore files for a workspace.
type GitignoreMatcher struct {
	ignores []*gitignore.GitIgnore
}

// LoadGitignore walks from workspace root and loads .gitignore files.
func LoadGitignore(workspace string) (*GitignoreMatcher, error) {
	m := &GitignoreMatcher{}
	err := filepath.WalkDir(workspace, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == ".gitignore" {
			gi, err := gitignore.CompileIgnoreFile(path)
			if err != nil {
				return err
			}
			m.ignores = append(m.ignores, gi)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Ignored reports whether rel path (slash-separated from workspace root) should be skipped.
func (m *GitignoreMatcher) Ignored(rel string) bool {
	if m == nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	for _, gi := range m.ignores {
		if gi.MatchesPath(rel) {
			return true
		}
	}
	// Skip hidden dirs except .ds-code
	base := filepath.Base(rel)
	if strings.HasPrefix(base, ".") && base != ".ds-code" {
		return true
	}
	return false
}
