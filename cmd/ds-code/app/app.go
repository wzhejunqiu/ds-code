package app

import (
	"github.com/hejunqiu/ds-code/internal/checkpoint"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/lsp"
	mcpsvc "github.com/hejunqiu/ds-code/internal/mcp"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/session/subagentstore"
	sessionsqlite "github.com/hejunqiu/ds-code/internal/session/sqlite"
	"github.com/hejunqiu/ds-code/internal/shelljobs/manager"
)

// App holds CLI runtime state and lazy-initialized dependencies.
type App struct {
	Cfg          *config.Config
	store        session.Store
	subStore     subagentstore.Store
	sqliteDB     *sessionsqlite.Store
	mcpMgr       *mcpsvc.Manager
	lspMgr       *lsp.Manager
	checkpointSt *checkpoint.Store
	shellJobs    *manager.Manager
}

// New builds an App from loaded configuration.
func New(cfg *config.Config) *App {
	return &App{Cfg: cfg}
}

func (a *App) openStore() (session.Store, error) {
	if a.store != nil {
		return a.store, nil
	}
	sqlite, err := sessionsqlite.OpenDefault(a.Cfg.ProjectRoot)
	if err != nil {
		return nil, err
	}
	a.sqliteDB = sqlite
	a.subStore = sqlite.SubagentStore()
	a.store = session.NewLazyStore(sqlite)
	return a.store, nil
}

func (a *App) openSubagentStore() (subagentstore.Store, error) {
	if _, err := a.openStore(); err != nil {
		return nil, err
	}
	return a.subStore, nil
}

func (a *App) openCheckpointStore() (*checkpoint.Store, error) {
	if a.checkpointSt != nil {
		return a.checkpointSt, nil
	}
	st, err := checkpoint.OpenStore(a.Cfg.ProjectRoot)
	if err != nil {
		return nil, err
	}
	a.checkpointSt = st
	return st, nil
}

func (a *App) closeMCP() {
	if a.mcpMgr != nil {
		_ = a.mcpMgr.Close()
		a.mcpMgr = nil
	}
}

func (a *App) closeStore() {
	if a.sqliteDB != nil {
		_ = a.sqliteDB.Close()
		a.sqliteDB = nil
		a.store = nil
		a.subStore = nil
	}
}
