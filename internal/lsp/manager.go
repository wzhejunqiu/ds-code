package lsp

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/lsp/client"
)

// Manager caches language server clients per server ID.
type Manager struct {
	root     string
	cfg      config.LSPConfig
	registry map[string]ServerConfig

	mu      sync.Mutex
	clients map[string]*client.Client
}

// NewManager creates an LSP manager for a project root.
func NewManager(root string, cfg config.LSPConfig) *Manager {
	return &Manager{
		root:     root,
		cfg:      cfg,
		registry: BuildRegistry(cfg),
		clients:  make(map[string]*client.Client),
	}
}

// Close shuts down all clients.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var first error
	for id, c := range m.clients {
		if err := c.Close(); err != nil && first == nil {
			first = err
		}
		delete(m.clients, id)
	}
	return first
}

// EnsureClient returns a started client for serverID.
func (m *Manager) EnsureClient(ctx context.Context, serverID string) (*client.Client, error) {
	m.mu.Lock()
	if c, ok := m.clients[serverID]; ok {
		m.mu.Unlock()
		return c, nil
	}
	srv, ok := m.registry[serverID]
	if !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("lsp server %q not configured", serverID)
	}
	if srv.Disabled {
		m.mu.Unlock()
		return nil, fmt.Errorf("lsp server %q is disabled", serverID)
	}
	if srv.Command == "" {
		m.mu.Unlock()
		return nil, fmt.Errorf("lsp server %q has no command configured (install the language server or set lsp.servers.%s.command)", serverID, serverID)
	}
	if _, err := exec.LookPath(srv.Command); err != nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("lsp server %q: %s not found in PATH", serverID, srv.Command)
	}
	m.mu.Unlock()

	c := client.NewClient(m.root, m.cfg, srv)
	if err := c.Start(ctx); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.clients[serverID]; ok {
		_ = c.Close()
		return existing, nil
	}
	m.clients[serverID] = c
	go m.idleWatch(serverID, c)
	return c, nil
}

func (m *Manager) idleWatch(serverID string, c *client.Client) {
	idle := m.cfg.IdleShutdown
	if idle <= 0 {
		idle = 120 * time.Second
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if time.Since(c.LastUsedAt()) < idle {
			continue
		}
		m.mu.Lock()
		if cur, ok := m.clients[serverID]; ok && cur == c {
			_ = cur.Close()
			delete(m.clients, serverID)
		}
		m.mu.Unlock()
		return
	}
}

// Registry returns the merged server registry.
func (m *Manager) Registry() map[string]ServerConfig {
	return m.registry
}

// StartedServerIDs returns server ids with active clients.
func (m *Manager) StartedServerIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.clients))
	for id := range m.clients {
		out = append(out, id)
	}
	return out
}
