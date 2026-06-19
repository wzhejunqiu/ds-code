package resultstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oklog/ulid/v2"
	"github.com/wzhejunqiu/ds-code/internal/datadir"
)

// Store persists MCP tool spill files under ~/.ds-code/projects/<id>/mcp-result/.
type Store struct {
	ProjectRoot string // cfg.ProjectRoot, not perm.Workspace
}

// Save writes full formatted tool body; creates parent dirs 0700, file 0600.
func (s *Store) Save(sessionID, callID, body string) (absPath string, err error) {
	if s == nil || s.ProjectRoot == "" {
		return "", fmt.Errorf("resultstore: store not configured")
	}
	if sessionID == "" {
		return "", fmt.Errorf("resultstore: empty session id")
	}
	stem := spillCallFilename(callID)
	dir, err := datadir.MCPResultSessionDir(s.ProjectRoot, sessionID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("resultstore: mkdir: %w", err)
	}
	absPath = filepath.Join(dir, stem+".txt")
	if err := os.WriteFile(absPath, []byte(body), 0o600); err != nil {
		return "", fmt.Errorf("resultstore: write: %w", err)
	}
	return absPath, nil
}

// spillCallFilename returns a filename-safe stem for mcp-result/<session>/<stem>.txt.
func spillCallFilename(rawID string) string {
	id := strings.TrimSpace(rawID)
	if id == "" {
		return ulid.Make().String()
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", "..", "_", "\x00", "")
	return replacer.Replace(id)
}
