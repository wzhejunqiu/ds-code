package app

import (
	"context"
	"fmt"

	"github.com/wzhejunqiu/ds-code/cmd/ds-code/slashcmd"
	"github.com/wzhejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/lsp"
	mcpsvc "github.com/wzhejunqiu/ds-code/internal/mcp"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/shelljobs/manager"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	toolsetup "github.com/wzhejunqiu/ds-code/internal/tool/setup"
	"go.uber.org/zap"
)

type toolBundle struct {
	reg    *tool.Registry
	lspMgr *lsp.Manager
	deps   toolsetup.Deps
}

func (a *App) buildTools(ctx context.Context, perm *permission.Engine, gi *tool.GitignoreMatcher, strict bool, llmClient llm.Client, runMode string) (*toolBundle, error) {
	var lspMgr *lsp.Manager
	if a.Cfg.LSP.Enabled {
		if a.lspMgr == nil {
			a.lspMgr = lsp.NewManager(a.Cfg.ProjectRoot, a.Cfg.LSP)
		}
		lspMgr = a.lspMgr
		for _, id := range a.Cfg.LSP.WarmupOnStart {
			if _, err := lspMgr.EnsureClient(ctx, id); err != nil {
				// best-effort warmup
				_ = err
			}
		}
	}

	shellMgr, err := a.openShellJobs()
	if err != nil {
		return nil, err
	}

	subStore, err := a.openSubagentStore()
	if err != nil {
		return nil, err
	}
	deps := toolsetup.Deps{
		Cfg:       a.Cfg,
		Perm:      perm,
		Gitignore: gi,
		Strict:    strict,
		LLM:       llmClient,
		LSP:       lspMgr,
		MCP:       a.mcpMgr,
		ShellJobs: shellMgr,
		Subagent:  subStore,
	}
	reg := toolsetup.BuildRegistry(runMode, deps)
	return &toolBundle{reg: reg, lspMgr: lspMgr, deps: deps}, nil
}

func (a *App) openShellJobs() (*manager.Manager, error) {
	if a.shellJobs != nil {
		return a.shellJobs, nil
	}
	mgr, err := manager.Open(a.Cfg.ProjectRoot, a.Cfg.Tools.Shell)
	if err != nil {
		return nil, err
	}
	a.shellJobs = mgr
	return mgr, nil
}

func (a *App) closeShellJobs() {
	if a.shellJobs != nil {
		a.shellJobs.Close()
		a.shellJobs = nil
	}
}

func (a *App) closeLSP() {
	if a.lspMgr != nil {
		_ = a.lspMgr.Close()
		a.lspMgr = nil
	}
}

func (a *App) rebindRunnerTools(runner *agent.Runner, ctxSvc *ctxpkg.Service, bundle *toolBundle) {
	runner.Tools = bundle.reg
	ctxSvc.Tools = bundle.reg
}

// SetRunMode implements slashcmd.Host.
func (a *App) SetRunMode(ctx context.Context, env *slashcmd.Env, mode string) error {
	env.Cfg.RunMode = mode
	if err := env.Store.UpdateSession(ctx, *env.SessionID, func(s *session.Session) error {
		s.RunMode = mode
		return nil
	}); err != nil {
		return err
	}
	gi, _ := tool.LoadGitignore(env.Cfg.ProjectRoot)
	bundle, err := a.buildTools(ctx, env.Runner.Perm, gi, env.Cfg.LLM.StrictTools, env.Runner.LLM, mode)
	if err != nil {
		return err
	}
	a.rebindRunnerTools(env.Runner, env.CtxSvc, bundle)
	fmt.Fprintf(env.Out, "Run mode set to %s (tools updated for this session).\n", mode)
	return nil
}

func (a *App) attachMCP(ctx context.Context, strict bool) error {
	if len(a.Cfg.MCP.Servers) == 0 {
		return nil
	}
	if a.mcpMgr == nil {
		mgr, err := mcpsvc.NewManagerFromConfig(ctx, a.Cfg.MCP.Servers, strict, a.Cfg.Tools.DeferMCP, a.Cfg.Tools.Shell.EnvBlacklistCompiled)
		if err != nil {
			return err
		}
		a.mcpMgr = mgr
	}
	return nil
}

func (a *App) ensureMCP(ctx context.Context, perm *permission.Engine, strict bool) error {
	if len(a.Cfg.MCP.Servers) == 0 {
		logging.L().Debug("MCP disabled (no servers configured)")
		return nil
	}
	logging.L().Info("connecting MCP servers", zap.Int("count", len(a.Cfg.MCP.Servers)))
	if err := a.attachMCP(ctx, strict); err != nil {
		logging.L().Error("MCP connect failed", zap.Error(err))
		return err
	}
	if a.mcpMgr != nil {
		perm.SetWriteToolDetector(a.mcpMgr.IsWriteTool)
		logging.L().Info("MCP ready")
	}
	return nil
}
