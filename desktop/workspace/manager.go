package workspace

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/cmd/ds-code/app"
	"github.com/wzhejunqiu/ds-code/cmd/ds-code/slashcmd"
	desktopbridge "github.com/wzhejunqiu/ds-code/desktop/bridge"
	desktopdatadir "github.com/wzhejunqiu/ds-code/desktop/datadir"
	desktopperm "github.com/wzhejunqiu/ds-code/desktop/permission"
	desktopsys "github.com/wzhejunqiu/ds-code/desktop/sys"
	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
	"github.com/wzhejunqiu/ds-code/internal/session"
)

// EmitFunc sends an agent event envelope to the UI.
type EmitFunc func(env desktopbridge.AgentEventEnvelope)

// Runtime holds lazily initialized backend state for one workspace.
type Runtime struct {
	app    *app.App
	runner *agent.Runner
	store  session.Store
	ctxSvc *ctxpkg.Service
	perm   *desktopperm.Registry
}

// Manager coordinates multiple workspaces and their agent runtimes.
type Manager struct {
	mu       sync.Mutex
	registry *Registry
	runtime  map[string]*Runtime
	turns    map[string]*turnState
	emit     EmitFunc
	emitSeq  uint64
}

// NewManager loads the registry and prepares an empty manager.
func NewManager(emit EmitFunc) (*Manager, error) {
	reg, err := LoadRegistry()
	if err != nil {
		return nil, err
	}
	return NewManagerWithRegistry(reg, emit)
}

// NewManagerWithRegistry constructs a manager with an explicit registry (for tests).
func NewManagerWithRegistry(reg *Registry, emit EmitFunc) (*Manager, error) {
	_ = billing.SetupFromUserConfig()
	return &Manager{
		registry: reg,
		runtime:  make(map[string]*Runtime),
		turns:    make(map[string]*turnState),
		emit:     emit,
	}, nil
}

// Registry returns the underlying registry (read-only use; call Save after mutations).
func (m *Manager) Registry() *Registry {
	return m.registry
}

func (m *Manager) turnState(wsID string) *turnState {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.turns[wsID]; ok {
		return s
	}
	s := &turnState{}
	m.turns[wsID] = s
	return s
}

func (m *Manager) permReg(wsID string) *desktopperm.Registry {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rt, ok := m.runtime[wsID]; ok && rt != nil {
		return rt.perm
	}
	return nil
}

func (m *Manager) emitFor(wsID string) EmitFunc {
	return func(env desktopbridge.AgentEventEnvelope) {
		env.WorkspaceID = wsID
		m.mu.Lock()
		m.emitSeq++
		env.Seq = m.emitSeq
		emit := m.emit
		m.mu.Unlock()
		if emit != nil {
			emit(env)
		}
	}
}

func (m *Manager) emitPermission(wsID string, p desktopbridge.PermissionRequestPayload) {
	m.turnState(wsID).setWaitingPerm(true)
	desktopsys.PermissionWaiting(true)
	m.emitFor(wsID)(desktopbridge.AgentEventEnvelope{
		V:           desktopbridge.EnvelopeVersion,
		StreamID:    "main",
		WorkspaceID: wsID,
		Kind:        desktopbridge.KindPermissionRequest,
		Ts:          time.Now().UnixMilli(),
		Critical:    true,
		Payload:     desktopbridge.MustPayload(p),
	})
}

// List returns workspace summaries for the sidebar.
func (m *Manager) List() []Summary {
	data := m.registry.Data()
	active := data.Active
	out := make([]Summary, 0, len(data.Workspaces))
	for _, e := range data.Workspaces {
		valid := true
		if st, err := os.Stat(e.Root); err != nil || !st.IsDir() {
			valid = false
		}
		out = append(out, Summary{
			ID:           e.ID,
			Name:         e.Name,
			Root:         e.Root,
			Active:       e.ID == active,
			LastOpenedAt: e.LastOpenedAt,
			Valid:        valid,
		})
	}
	return out
}

// ActiveID returns the active workspace id.
func (m *Manager) ActiveID() string {
	return m.registry.Active()
}

// Switch sets the active workspace and touches last-opened.
func (m *Manager) Switch(id string) error {
	found := false
	for _, e := range m.registry.Data().Workspaces {
		if e.ID == id {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("workspace not found: %s", id)
	}
	m.registry.SetActive(id)
	for i := range m.registry.data.Workspaces {
		if m.registry.data.Workspaces[i].ID == id {
			m.registry.data.Workspaces[i].LastOpenedAt = time.Now().Unix()
			break
		}
	}
	return m.registry.Save()
}

// Add opens or activates a workspace by project root path.
func (m *Manager) Add(root string) (Summary, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Summary{}, err
	}
	st, err := os.Stat(abs)
	if err != nil || !st.IsDir() {
		return Summary{}, fmt.Errorf("project root not found: %s", abs)
	}
	id := desktopdatadir.ProjectID(abs)
	name := filepath.Base(abs)
	m.registry.Upsert(Entry{
		ID:   id,
		Root: abs,
		Name: name,
	})
	m.registry.SetActive(id)
	if err := m.registry.Save(); err != nil {
		return Summary{}, err
	}
	if _, err := m.initRuntime(id); err != nil {
		return Summary{}, err
	}
	return Summary{
		ID:           id,
		Name:         name,
		Root:         abs,
		Active:       true,
		LastOpenedAt: time.Now().Unix(),
		Valid:        true,
	}, nil
}

// Remove removes a workspace from the registry (does not delete project data).
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	if rt, ok := m.runtime[id]; ok && rt != nil && rt.app != nil {
		rt.app.CloseDesktop()
		delete(m.runtime, id)
	}
	delete(m.turns, id)
	m.mu.Unlock()

	if !m.registry.Remove(id) {
		return fmt.Errorf("workspace not found: %s", id)
	}
	return m.registry.Save()
}

// Ensure lazily initializes the workspace runtime.
func (m *Manager) Ensure(id string) (*Runtime, error) {
	m.mu.Lock()
	if rt, ok := m.runtime[id]; ok && rt != nil && rt.runner != nil {
		m.mu.Unlock()
		return rt, nil
	}
	m.mu.Unlock()
	return m.initRuntime(id)
}

func (m *Manager) initRuntime(id string) (*Runtime, error) {
	var entry *Entry
	for i := range m.registry.data.Workspaces {
		if m.registry.data.Workspaces[i].ID == id {
			entry = &m.registry.data.Workspaces[i]
			break
		}
	}
	if entry == nil {
		return nil, fmt.Errorf("workspace not found: %s", id)
	}

	cfg, err := config.Load(nil, config.Options{
		StartDir:           entry.Root,
		RequireAPIKey:      false,
		SkipProjectDataDir: true,
	})
	if err != nil {
		return nil, err
	}
	dir, err := desktopdatadir.EnsureProjectDataDir(cfg.ProjectRoot)
	if err != nil {
		return nil, err
	}
	cfg.ProjectDataDir = dir

	wsID := id
	permReg := desktopperm.NewRegistry(func(p desktopbridge.PermissionRequestPayload) {
		m.emitPermission(wsID, p)
	})

	a := app.New(cfg)
	runner, store, ctxSvc, err := a.NewDesktopRunner(io.Discard, permReg.Prompter())
	if err != nil {
		a.CloseDesktop()
		return nil, err
	}
	runner.Perm.Interactive = true
	if cfg.Permission.Mode == permissionmode.Ask {
		runner.Perm.Prompter = permReg.Prompter()
	}
	runner.Perm.WebFetchPrompter = permReg.WebFetchPrompter()

	rt := &Runtime{app: a, runner: runner, store: store, ctxSvc: ctxSvc, perm: permReg}
	m.mu.Lock()
	if old, ok := m.runtime[id]; ok && old != nil && old.app != nil && old.app != a {
		old.app.CloseDesktop()
	}
	m.runtime[id] = rt
	m.mu.Unlock()
	return rt, nil
}

// RuntimeFor returns initialized runtime or nil.
func (m *Manager) RuntimeFor(id string) (*Runtime, error) {
	return m.Ensure(id)
}

// PermissionRegistry returns the permission registry for a workspace.
func (m *Manager) PermissionRegistry(id string) (*desktopperm.Registry, error) {
	rt, err := m.Ensure(id)
	if err != nil {
		return nil, err
	}
	return rt.perm, nil
}

// Close shuts down all workspace runtimes.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, rt := range m.runtime {
		if rt != nil && rt.app != nil {
			rt.app.CloseDesktop()
		}
		delete(m.runtime, id)
	}
}

// SaveRegistry persists registry changes.
func (m *Manager) SaveRegistry() error {
	return m.registry.Save()
}

// CreateChat creates a new agent conversation in a workspace.
func (m *Manager) CreateChat(wsID string) (ChatSummary, error) {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return ChatSummary{}, err
	}
	sess, err := slashcmd.CreateSession(rt.app.Cfg, rt.store)
	if err != nil {
		return ChatSummary{}, err
	}
	ctx := context.Background()
	defaultFmt := DefaultAssistantOutputFormat()
	if err := rt.store.UpdateSession(ctx, sess.ID, func(s *session.Session) error {
		s.AssistantOutputFormat = defaultFmt
		return nil
	}); err != nil {
		return ChatSummary{}, err
	}
	if err := slashcmd.SeedGitSnapshot(rt.app.Cfg, ctx, rt.store, sess.ID); err != nil {
		return ChatSummary{}, err
	}
	return chatSummaryFromSession(sess), nil
}

func chatSummaryFromSession(s session.Session) ChatSummary {
	title := s.Title
	if title == "" {
		title = "(untitled)"
	}
	return ChatSummary{
		ID:        s.ID,
		Title:     title,
		Model:     s.Model,
		RunMode:   s.RunMode.String(),
		UpdatedAt: s.UpdatedAt.UnixMilli(),
		CreatedAt: s.CreatedAt.UnixMilli(),
	}
}
