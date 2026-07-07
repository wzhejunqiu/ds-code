package app

import (
	"context"
	"os/exec"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

// MCPServerStatusView is one MCP server row for the desktop settings UI.
type MCPServerStatusView struct {
	Name            string `json:"name"`
	Command         string `json:"command"`
	Connected       bool   `json:"connected"`
	RegisteredTools int    `json:"registeredTools"`
	DiscoveredTools int    `json:"discoveredTools"`
}

// MCPSkippedToolView is a skipped MCP tool row.
type MCPSkippedToolView struct {
	Server string `json:"server"`
	Tool   string `json:"tool"`
	Reason string `json:"reason"`
}

// MCPStatusView summarizes MCP configuration and runtime state.
type MCPStatusView struct {
	ConfiguredServers []MCPServerStatusView `json:"configuredServers"`
	SkippedTools      []MCPSkippedToolView  `json:"skippedTools"`
	Connected         bool                  `json:"connected"`
}

// LSPServerStatusView is one LSP server row for the desktop settings UI.
type LSPServerStatusView struct {
	ID           string `json:"id"`
	Command      string `json:"command"`
	Disabled     bool   `json:"disabled"`
	CommandFound bool   `json:"commandFound"`
	Started      bool   `json:"started"`
	Hint         string `json:"hint,omitempty"`
}

// LSPStatusView summarizes LSP configuration and runtime state.
type LSPStatusView struct {
	Enabled bool                  `json:"enabled"`
	Servers []LSPServerStatusView `json:"servers"`
}

// ServiceStatusView combines MCP and LSP status for Inspector overview.
type ServiceStatusView struct {
	MCP MCPStatusView `json:"mcp"`
	LSP LSPStatusView `json:"lsp"`
}

// MCPLSPConfigView is editable MCP/LSP config for settings UI.
type MCPLSPConfigView struct {
	MCP config.MCPConfig `json:"mcp"`
	LSP config.LSPConfig `json:"lsp"`
}

// MCPLSPConfig returns MCP/LSP sections from loaded config.
func MCPLSPConfig(cfg *config.Config) MCPLSPConfigView {
	return MCPLSPConfigView{MCP: cfg.MCP, LSP: cfg.LSP}
}

// MCPStatus returns MCP configuration and connection status.
func (a *App) MCPStatus() MCPStatusView {
	view := MCPStatusView{}
	connected := map[string]struct{}{}
	if a.mcpMgr != nil {
		view.Connected = true
		for _, n := range a.mcpMgr.ServerNames() {
			connected[n] = struct{}{}
		}
	}
	regTools, discTools := 0, 0
	if a.mcpMgr != nil {
		regTools = a.mcpMgr.RegisteredToolCount()
		discTools = a.mcpMgr.ToolCount()
		for _, sk := range a.mcpMgr.SkippedTools() {
			view.SkippedTools = append(view.SkippedTools, MCPSkippedToolView{
				Server: sk.Server,
				Tool:   sk.Tool,
				Reason: string(sk.Reason),
			})
		}
	}
	for _, s := range a.Cfg.MCP.Servers {
		row := MCPServerStatusView{
			Name:            s.Name,
			Command:         s.Command,
			RegisteredTools: regTools,
			DiscoveredTools: discTools,
		}
		if _, ok := connected[s.Name]; ok {
			row.Connected = true
		}
		view.ConfiguredServers = append(view.ConfiguredServers, row)
	}
	return view
}

// LSPStatus returns LSP configuration and warmup state.
func (a *App) LSPStatus() LSPStatusView {
	view := LSPStatusView{Enabled: a.Cfg.LSP.Enabled}
	started := map[string]struct{}{}
	if a.lspMgr != nil {
		for _, id := range a.lspMgr.StartedServerIDs() {
			started[id] = struct{}{}
		}
	}
	for id, s := range a.Cfg.LSP.Servers {
		row := LSPServerStatusView{
			ID:       id,
			Command:  s.Command,
			Disabled: s.Disabled,
		}
		if s.Command != "" {
			if _, err := exec.LookPath(s.Command); err == nil {
				row.CommandFound = true
			} else {
				row.Hint = "Install " + s.Command + " or set lsp.servers." + id + ".command in config"
			}
		} else if !s.Disabled {
			row.Hint = "Set lsp.servers." + id + ".command in config"
		}
		if _, ok := started[id]; ok {
			row.Started = true
		}
		view.Servers = append(view.Servers, row)
	}
	return view
}

// ServiceStatus returns combined MCP/LSP status.
func (a *App) ServiceStatus() ServiceStatusView {
	return ServiceStatusView{
		MCP: a.MCPStatus(),
		LSP: a.LSPStatus(),
	}
}

// ReloadMCPLSPFromRunner closes and reconnects MCP/LSP, then rebinds runner tools.
func (a *App) ReloadMCPLSPFromRunner(ctx context.Context, runner *agent.Runner, ctxSvc *ctxpkg.Service) error {
	a.closeMCP()
	a.closeLSP()
	if err := a.ensureMCP(ctx, runner.Perm, a.Cfg.LLM.StrictTools); err != nil {
		return err
	}
	searchSkip := searchskip.New(a.Cfg.Tools.Search.SkipDirs)
	bundle, err := a.buildTools(ctx, runner.Perm, searchSkip, a.Cfg.LLM.StrictTools, runner.LLM, a.Cfg.RunMode)
	if err != nil {
		return err
	}
	a.rebindRunnerTools(runner, ctxSvc, bundle)
	a.logMCPSkipped()
	return nil
}
