package permission

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
)

// resolveMCPSpillRead allows read_file on spill files for the current session only.
func (e *Engine) resolveMCPSpillRead(rel string) (string, bool) {
	if e.ProjectRoot == "" || e.SpillSessionID == "" {
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
	prefix := filepath.Join(dataDir, "mcp-result", e.SpillSessionID) + string(filepath.Separator)
	if !strings.HasPrefix(abs+string(filepath.Separator), prefix) {
		return "", false
	}
	info, err := os.Stat(abs)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	if !strings.HasSuffix(abs, ".txt") {
		return "", false
	}
	return abs, true
}
