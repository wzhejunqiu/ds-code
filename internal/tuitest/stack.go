//go:build tuitest

package tuitest

import (
	"io"
	"os"
	"testing"

	"github.com/wzhejunqiu/ds-code/cmd/ds-code/app"
	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
	"github.com/wzhejunqiu/ds-code/internal/tuitest/mockserver"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
)

// Stack wires production app/runner/TUI deps against a mock LLM server.
type Stack struct {
	App      *app.App
	Runner   *agent.Runner
	Store    session.Store
	CtxSvc   *ctxpkg.Service
	Cfg      *config.Config
	Mock     *mockserver.Server
	Registry *mockserver.Registry
	Project  string
	homeDir  string // isolated HOME; removed on Close (see testutil.NewIsolatedHome)
	cleanup  []func()
}

// NewHarness creates a stack for the interactive ds-code-tui-test binary.
// All ~/.ds-code data is rooted under testutil.NewIsolatedHome and removed on Close.
func NewHarness() (*Stack, error) {
	homeDir, err := testutil.NewIsolatedHome()
	if err != nil {
		return nil, err
	}
	s, err := newStackCore()
	if err != nil {
		_ = os.RemoveAll(homeDir)
		return nil, err
	}
	s.homeDir = homeDir
	return s, nil
}

// NewStack creates an isolated harness stack for tests.
func NewStack(t testing.TB) (*Stack, error) {
	t.Helper()
	testutil.IsolatedHome(t)
	s, err := newStackCore()
	if err != nil {
		return nil, err
	}
	t.Cleanup(s.Close)
	return s, nil
}

func newStackCore() (*Stack, error) {
	dir, err := os.MkdirTemp("", "ds-code-tui-test-*")
	if err != nil {
		return nil, err
	}

	if err := PrepareProjectRoot(dir); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}

	reg := mockserver.NewRegistry()
	mock := mockserver.New(reg)

	cfg, err := HarnessConfig(mock.BaseURL(), dir)
	if err != nil {
		mock.Close()
		_ = os.RemoveAll(dir)
		return nil, err
	}

	application := app.New(cfg)
	runner, store, ctxSvc, err := application.NewRunner(io.Discard)
	if err != nil {
		mock.Close()
		application.Close()
		_ = os.RemoveAll(dir)
		return nil, err
	}

	s := &Stack{
		App:      application,
		Runner:   runner,
		Store:    store,
		CtxSvc:   ctxSvc,
		Cfg:      cfg,
		Mock:     mock,
		Registry: reg,
		Project:  dir,
	}
	s.cleanup = []func(){
		func() { mock.Close() },
		func() { application.Close() },
	}
	return s, nil
}

// Close releases resources and removes harness temp directories.
func (s *Stack) Close() {
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		s.cleanup[i]()
	}
	s.removeHarnessDirs()
}

func (s *Stack) removeHarnessDirs() {
	if s.Project != "" {
		_ = os.RemoveAll(s.Project)
		s.Project = ""
	}
	if s.homeDir != "" {
		_ = os.RemoveAll(s.homeDir)
		s.homeDir = ""
		return
	}
	if s.Cfg != nil && s.Cfg.ProjectDataDir != "" {
		_ = os.RemoveAll(s.Cfg.ProjectDataDir)
	}
}

// Deps builds tui.Deps; sessionID must already exist.
func (s *Stack) Deps(sessionID string, handleSlash deps.SlashFunc) deps.Deps {
	subStore, _ := s.App.OpenSubagentStore()
	return deps.Deps{
		Cfg:         s.Cfg,
		Runner:      s.Runner,
		Store:       s.Store,
		Subagent:    subStore,
		Context:     s.CtxSvc,
		SessionID:   sessionID,
		Version:     "tui-test",
		HandleSlash: handleSlash,
	}
}
