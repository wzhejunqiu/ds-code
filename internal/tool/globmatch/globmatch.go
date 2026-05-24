package globmatch

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/permission"
)

// HasMeta reports whether path contains glob metacharacters.
func HasMeta(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

// SplitPath splits a relative path into a directory base and glob pattern.
// When path has no metacharacters, pattern is empty.
func SplitPath(path string) (base, pattern string) {
	if path == "" || path == "." {
		return ".", ""
	}
	i := strings.IndexAny(path, "*?[")
	if i < 0 {
		return path, ""
	}
	slash := strings.LastIndex(path[:i], "/")
	if slash < 0 {
		return ".", path
	}
	return path[:slash], path[slash+1:]
}

// MatchFiles returns absolute paths under root matching pattern.
// limit <= 0 means no limit on the number of paths returned.
func MatchFiles(root, pattern string, limit int) ([]string, error) {
	if strings.Contains(pattern, "**") {
		return matchDoubleStar(root, pattern, limit)
	}
	matches, err := filepath.Glob(filepath.Join(root, pattern))
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func matchDoubleStar(root, pattern string, limit int) ([]string, error) {
	suffix := strings.TrimPrefix(pattern, "**/")
	if suffix == pattern {
		suffix = strings.TrimPrefix(pattern, "**")
	}
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
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
		base := filepath.Base(path)
		ok := suffix == "" || strings.HasSuffix(path, suffix)
		if !ok && suffix != "" {
			ok, _ = filepath.Match(suffix, base)
		}
		if ok {
			out = append(out, path)
		}
		if limit > 0 && len(out) >= limit {
			return filepath.SkipAll
		}
		return nil
	})
	return out, err
}
