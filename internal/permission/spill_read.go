package permission

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
)

// resolveProjectDataRead allows read_file on regular files under the current
// project's data directory (~/.ds-code/projects/<project_id>/).
func (e *Engine) resolveProjectDataRead(rel string) (string, bool) {
	if e.ProjectRoot == "" {
		return "", false
	}
	if !filepath.IsAbs(rel) {
		return "", false
	}
	abs := filepath.Clean(rel)
	dataDir, err := datadir.ProjectDataDir(e.ProjectRoot)
	if err != nil {
		return "", false
	}
	prefix := dataDir + string(filepath.Separator)
	if !strings.HasPrefix(abs+string(filepath.Separator), prefix) {
		return "", false
	}
	info, err := os.Stat(abs)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	return abs, true
}

// IsProjectDataPath reports whether abs is a regular file under the current
// project's data directory. abs must already be cleaned absolute path.
func (e *Engine) IsProjectDataPath(abs string) bool {
	got, ok := e.resolveProjectDataRead(abs)
	return ok && got == filepath.Clean(abs)
}
