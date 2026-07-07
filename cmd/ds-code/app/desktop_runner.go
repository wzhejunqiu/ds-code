package app

import (
	"io"

	desktopdatadir "github.com/wzhejunqiu/ds-code/desktop/datadir"
	"github.com/wzhejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

// NewDesktopRunner builds a runner using desktop-isolated data paths under
// ~/.ds-code/desktop/projects/<project-id>/.
func (a *App) NewDesktopRunner(out io.Writer, prompter permission.Prompter) (*agent.Runner, session.Store, *ctxpkg.Service, error) {
	a.useDesktopDataDir = true
	runner, store, ctxSvc, err := a.newRunner(out)
	if err != nil {
		return nil, nil, nil, err
	}
	if prompter != nil {
		runner.Perm.Interactive = true
		runner.Perm.Prompter = prompter
	}
	if dir, err := desktopdatadir.EnsureProjectDataDir(a.Cfg.ProjectRoot); err == nil {
		a.Cfg.ProjectDataDir = dir
	}
	return runner, store, ctxSvc, nil
}

// OpenSubagentStoreForDesktop exposes subagent store for desktop bindings.
func (a *App) OpenSubagentStoreForDesktop() (subagentstore.Store, error) {
	a.useDesktopDataDir = true
	return a.openSubagentStore()
}

// CloseDesktop releases desktop app resources.
func (a *App) CloseDesktop() {
	a.closeStore()
	a.closeMCP()
	a.closeLSP()
	a.closeShellJobs()
}
