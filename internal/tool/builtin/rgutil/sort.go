package rgutil

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/permission"
)

// FileModTime returns mod time for rel under perm.Workspace.
func FileModTime(perm *permission.Engine, rel string) time.Time {
	abs := filepath.Join(perm.Workspace, filepath.FromSlash(rel))
	info, err := os.Stat(abs)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// SortFilesByMtime sorts files by modification time descending, then path ascending.
func SortFilesByMtime(files []string, perm *permission.Engine) {
	sort.Slice(files, func(i, j int) bool {
		ti := FileModTime(perm, files[i])
		tj := FileModTime(perm, files[j])
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return files[i] < files[j]
	})
}
