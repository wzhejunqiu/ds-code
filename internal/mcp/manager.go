package mcp

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"go.uber.org/zap"
)

// Manager connects configured MCP servers and exposes their tools.
type Manager struct {
	servers  []*Server
	tools    []*adapterTool
	byName   map[string]*adapterTool
	deferMCP bool
	mu       sync.RWMutex
}

// NewManager connects all configured MCP servers and discovers tools.
func NewManager(ctx context.Context, cfgs []config.MCPServerConfig, envBlacklist []*regexp.Regexp) (*Manager, error) {
	if len(cfgs) == 0 {
		return &Manager{byName: make(map[string]*adapterTool)}, nil
	}

	names := make(map[string]struct{})
	m := &Manager{byName: make(map[string]*adapterTool)}

	for _, cfg := range cfgs {
		if err := ValidateServerName(cfg.Name); err != nil {
			m.Close()
			return nil, err
		}
		if _, dup := names[cfg.Name]; dup {
			m.Close()
			return nil, fmt.Errorf("mcp: duplicate server name %q", cfg.Name)
		}
		names[cfg.Name] = struct{}{}

		srv, err := ConnectServer(ctx, cfg, envBlacklist)
		if err != nil {
			m.Close()
			return nil, err
		}
		m.servers = append(m.servers, srv)
	}

	return m, nil
}

// NewManagerFromConfig connects servers and discovers tools.
func NewManagerFromConfig(ctx context.Context, cfgs []config.MCPServerConfig, strict bool, deferMCP bool, envBlacklist []*regexp.Regexp) (*Manager, error) {
	m, err := NewManager(ctx, cfgs, envBlacklist)
	if err != nil {
		return nil, err
	}
	m.deferMCP = deferMCP
	if len(cfgs) == 0 {
		return m, nil
	}
	if err := m.DiscoverTools(ctx, strict); err != nil {
		m.Close()
		return nil, err
	}
	return m, nil
}

// DiscoverTools loads tool definitions from all servers (call after NewManager).
func (m *Manager) DiscoverTools(ctx context.Context, strict bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tools = nil
	m.byName = make(map[string]*adapterTool)

	for _, srv := range m.servers {
		list, err := srv.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("mcp: server %q list tools: %w", srv.Name, err)
		}
		logging.L().Debug("mcp discover tools",
			zap.String("server", srv.Name),
			zap.Int("tool_count", len(list)),
		)
		for _, t := range list {
			ad := newAdapterTool(srv, t, strict, m.deferMCP)
			if _, exists := m.byName[ad.Name()]; exists {
				return fmt.Errorf("mcp: duplicate tool name %q", ad.Name())
			}
			m.tools = append(m.tools, ad)
			m.byName[ad.Name()] = ad
		}
	}
	return nil
}

// Register adds all MCP tools to the tool registry.
func (m *Manager) Register(reg *tool.Registry) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.tools {
		reg.Register(t)
	}
}

// IsWriteTool implements permission write detection for normalized MCP tool names.
func (m *Manager) IsWriteTool(name string) bool {
	if !IsMCPTool(name) {
		return false
	}
	m.mu.RLock()
	ad, ok := m.byName[name]
	m.mu.RUnlock()
	if ok {
		switch ad.level {
		case permission.LevelHigh, permission.LevelHighest:
			return true
		default:
			return IsWriteTool(name)
		}
	}
	return IsWriteTool(name)
}

// ToolCount returns the number of registered MCP tools.
func (m *Manager) ToolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tools)
}

// ServerNames returns connected server ids.
func (m *Manager) ServerNames() []string {
	out := make([]string, len(m.servers))
	for i, s := range m.servers {
		out[i] = s.Name
	}
	return out
}

// Close stops all MCP server processes.
func (m *Manager) Close() error {
	var first error
	for _, s := range m.servers {
		if err := s.Close(); err != nil && first == nil {
			first = err
		}
	}
	m.servers = nil
	m.tools = nil
	m.byName = make(map[string]*adapterTool)
	return first
}
