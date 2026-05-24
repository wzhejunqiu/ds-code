//go:build tuitest

package app

import (
	"io"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

// NewRunner exposes production runner assembly for the TUI harness.
func (a *App) NewRunner(out io.Writer) (*agent.Runner, session.Store, *ctxpkg.Service, error) {
	return a.newRunner(out)
}

// OpenSubagentStore exposes the subagent store for TUI deps.
func (a *App) OpenSubagentStore() (subagentstore.Store, error) {
	return a.openSubagentStore()
}

// Close releases app resources (stores, MCP, LSP, shell jobs).
func (a *App) Close() {
	a.closeStore()
	a.closeMCP()
	a.closeLSP()
	a.closeShellJobs()
}
