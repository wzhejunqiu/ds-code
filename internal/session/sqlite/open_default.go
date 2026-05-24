package sqlite

import (
	"fmt"
	"os"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

// OpenDefault opens the project sessions.db (Phase 3+).
func OpenDefault(projectRoot string) (*Store, error) {
	path := config.DefaultDBPath(projectRoot)
	if path == "" {
		return nil, fmt.Errorf("session: invalid project root")
	}
	store, err := Open(path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil && !os.IsNotExist(err) {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}
