package workspace

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wzhejunqiu/ds-code/cmd/ds-code/app"
	"github.com/wzhejunqiu/ds-code/internal/config"
)

// ServiceStatusView is re-exported for Wails bindings.
type ServiceStatusView = app.ServiceStatusView

// MCPLSPConfigView is editable MCP/LSP config.
type MCPLSPConfigView struct {
	MCP config.MCPConfig `json:"mcp"`
	LSP config.LSPConfig `json:"lsp"`
}

// ServiceStatus returns MCP/LSP runtime status for a workspace.
func (m *Manager) ServiceStatus(wsID string) (ServiceStatusView, error) {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return ServiceStatusView{}, err
	}
	return rt.app.ServiceStatus(), nil
}

// GetMCPLSPConfig loads MCP/LSP config for user or project scope.
func (m *Manager) GetMCPLSPConfig(scope, wsID string) (MCPLSPConfigView, error) {
	var startDir string
	if scope == "project" {
		root, err := m.ProjectRoot(wsID)
		if err != nil {
			return MCPLSPConfigView{}, err
		}
		startDir = root
	}
	cfg, err := config.Load(nil, config.Options{
		StartDir:           startDir,
		RequireAPIKey:      false,
		SkipProjectDataDir: true,
	})
	if err != nil {
		return MCPLSPConfigView{}, err
	}
	return MCPLSPConfigView{MCP: cfg.MCP, LSP: cfg.LSP}, nil
}

// SaveMCPLSPConfig writes MCP/LSP config atomically.
func (m *Manager) SaveMCPLSPConfig(scope, wsID string, raw string) error {
	var view MCPLSPConfigView
	if err := json.Unmarshal([]byte(raw), &view); err != nil {
		return fmt.Errorf("invalid mcp/lsp config json: %w", err)
	}
	var projectRoot string
	isProject := scope == "project"
	if isProject {
		root, err := m.ProjectRoot(wsID)
		if err != nil {
			return err
		}
		projectRoot = root
	}
	if err := config.SaveMCPServers(projectRoot, isProject, view.MCP.Servers); err != nil {
		return err
	}
	return config.SaveLSPConfig(projectRoot, isProject, view.LSP)
}

// ReloadWorkspaceServices reconnects MCP/LSP for a workspace.
func (m *Manager) ReloadWorkspaceServices(wsID string) error {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return err
	}
	root, err := m.ProjectRoot(wsID)
	if err != nil {
		return err
	}
	cfg, err := config.Load(nil, config.Options{
		StartDir:           root,
		RequireAPIKey:      false,
		SkipProjectDataDir: true,
	})
	if err != nil {
		return err
	}
	rt.app.Cfg = cfg
	return rt.app.ReloadMCPLSPFromRunner(context.Background(), rt.runner, rt.ctxSvc)
}
