package mcp

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"go.uber.org/zap"
)

// Manager connects configured MCP servers and exposes their tools.
type Manager struct {
	servers         []*Server
	tools           []*adapterTool
	byName          map[string]*adapterTool
	skipped         []SkippedTool
	registeredCount int
	registeredNames map[string]struct{}
	deferMCP        bool
	mu              sync.RWMutex
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

// NewManagerWithServers returns a Manager wired to pre-built servers (for unit tests).
// Callers must run DiscoverTools before Register.
func NewManagerWithServers(servers ...*Server) *Manager {
	return &Manager{
		servers: servers,
		byName:  make(map[string]*adapterTool),
	}
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
	m.skipped = nil
	m.registeredCount = 0
	m.registeredNames = nil

	type serverTools struct {
		srv   *Server
		tools []mcpsdk.Tool
	}

	var all []serverTools
	serverSets := make(map[string]map[string]struct{})
	perServerCounts := make(map[string]map[string]int)

	for _, srv := range m.servers {
		list, err := srv.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("mcp: server %q list tools: %w", srv.Name, err)
		}
		logging.L().Debug("mcp discover tools",
			zap.String("server", srv.Name),
			zap.Int("tool_count", len(list)),
		)
		all = append(all, serverTools{srv: srv, tools: list})
		if perServerCounts[srv.Name] == nil {
			perServerCounts[srv.Name] = make(map[string]int)
		}
		for _, t := range list {
			name := t.Name
			if serverSets[name] == nil {
				serverSets[name] = make(map[string]struct{})
			}
			serverSets[name][srv.Name] = struct{}{}
			perServerCounts[srv.Name][name]++
		}
	}

	seenInServer := make(map[string]map[string]bool)

	for _, st := range all {
		srv := st.srv
		if seenInServer[srv.Name] == nil {
			seenInServer[srv.Name] = make(map[string]bool)
		}
		for _, t := range st.tools {
			name := t.Name
			if len(serverSets[name]) > 1 {
				m.recordSkipLocked(srv.Name, name, SkipCrossServerDuplicate)
				continue
			}
			if perServerCounts[srv.Name][name] > 1 {
				if seenInServer[srv.Name][name] {
					m.recordSkipLocked(srv.Name, name, SkipInServerDuplicate)
					continue
				}
				seenInServer[srv.Name][name] = true
			}
			ad := newAdapterTool(srv, t, strict, m.deferMCP)
			m.tools = append(m.tools, ad)
			m.byName[name] = ad
		}
	}
	return nil
}

// Register adds all MCP tools to the tool registry, skipping builtin conflicts.
func (m *Manager) Register(reg *tool.Registry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registeredCount = 0
	m.registeredNames = make(map[string]struct{})
	for _, ad := range m.tools {
		if _, exists := reg.Get(ad.Name()); exists {
			m.recordSkipLocked(ad.server.Name, ad.mcpTool, SkipBuiltinConflict)
			continue
		}
		reg.RegisterMCPTool(ad, ad.server.Name)
		m.registeredCount++
		m.registeredNames[ad.Name()] = struct{}{}
	}
}

// SkippedTools returns tools that were not registered and why.
func (m *Manager) SkippedTools() []SkippedTool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]SkippedTool, len(m.skipped))
	copy(out, m.skipped)
	return out
}

func (m *Manager) recordSkipLocked(server, tool string, reason SkipReason) {
	m.skipped = append(m.skipped, SkippedTool{
		Server: server,
		Tool:   tool,
		Reason: reason,
	})
}

// IsWriteTool reports whether a registered MCP bare name requires write permission.
func (m *Manager) IsWriteTool(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.registeredNames[name]; !ok {
		return false
	}
	ad, ok := m.byName[name]
	if !ok {
		return false
	}
	switch ad.level {
	case permission.LevelHigh, permission.LevelHighest:
		return true
	default:
		return isWriteMCPToolName(ad.mcpTool)
	}
}

// ToolCount returns discovered MCP tools (including those later skipped at Register).
func (m *Manager) ToolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tools)
}

// RegisteredToolCount returns MCP tools successfully registered in the last Register call.
func (m *Manager) RegisteredToolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registeredCount
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
	m.skipped = nil
	m.registeredCount = 0
	m.registeredNames = nil
	return first
}
