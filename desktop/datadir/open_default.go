package datadir

import (
	"fmt"
	"os"

	sessionsqlite "github.com/wzhejunqiu/ds-code/internal/session/sqlite"
)

// OpenDefault opens the desktop sessions.db for a project.
func OpenDefault(projectRoot string) (*sessionsqlite.Store, error) {
	path := DefaultDBPath(projectRoot)
	if path == "" {
		return nil, fmt.Errorf("desktop datadir: invalid project root")
	}
	if _, err := EnsureProjectDataDir(projectRoot); err != nil {
		return nil, err
	}
	store, err := sessionsqlite.Open(path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil && !os.IsNotExist(err) {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}
