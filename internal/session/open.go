package session

import (
	"fmt"
	"os"

	"github.com/hejunqiu/ds-code/internal/config"
)

// OpenDefaultStore opens the project sessions.db (Phase 3+).
func OpenDefaultStore(projectRoot string) (*SQLiteStore, error) {
	path := config.DefaultDBPath(projectRoot)
	if path == "" {
		return nil, fmt.Errorf("session: invalid project root")
	}
	store, err := OpenSQLite(path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil && !os.IsNotExist(err) {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}
