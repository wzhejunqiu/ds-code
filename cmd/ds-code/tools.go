package main

import (
	"context"

	"github.com/hejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/lsp"
	mcpsvc "github.com/hejunqiu/ds-code/internal/mcp"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/shelljobs"
	"github.com/hejunqiu/ds-code/internal/tool"
	toolsetup "github.com/hejunqiu/ds-code/internal/tool/setup"
)

type toolBundle struct {
	reg    *tool.Registry
	lspMgr *lsp.Manager
	deps   toolsetup.Deps
}

func (a *app) buildTools(ctx context.Context, perm *permission.Engine, gi *tool.GitignoreMatcher, strict bool, llmClient llm.Client, runMode string) (*toolBundle, error) {
	var lspMgr *lsp.Manager
	if a.cfg.LSP.Enabled {
		if a.lspMgr == nil {
			a.lspMgr = lsp.NewManager(a.cfg.ProjectRoot, a.cfg.LSP)
		}
		lspMgr = a.lspMgr
		for _, id := range a.cfg.LSP.WarmupOnStart {
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

	deps := toolsetup.Deps{
		Cfg:       a.cfg,
		Perm:      perm,
		Gitignore: gi,
		Strict:    strict,
		LLM:       llmClient,
		LSP:       lspMgr,
		MCP:       a.mcpMgr,
		ShellJobs: shellMgr,
	}
	reg := toolsetup.BuildRegistry(runMode, deps)
	return &toolBundle{reg: reg, lspMgr: lspMgr, deps: deps}, nil
}

func (a *app) openShellJobs() (*shelljobs.Manager, error) {
	if a.shellJobs != nil {
		return a.shellJobs, nil
	}
	mgr, err := shelljobs.OpenManager(a.cfg.ProjectRoot, a.cfg.Tools.Shell)
	if err != nil {
		return nil, err
	}
	a.shellJobs = mgr
	return mgr, nil
}

func (a *app) closeShellJobs() {
	if a.shellJobs != nil {
		a.shellJobs.Close()
		a.shellJobs = nil
	}
}

func (a *app) closeLSP() {
	if a.lspMgr != nil {
		_ = a.lspMgr.Close()
		a.lspMgr = nil
	}
}

func (a *app) rebindRunnerTools(runner *agent.Runner, ctxSvc *ctxpkg.Service, bundle *toolBundle) {
	runner.Tools = bundle.reg
	ctxSvc.Tools = bundle.reg
}

func (a *app) attachMCP(ctx context.Context, strict bool) error {
	if len(a.cfg.MCP.Servers) == 0 {
		return nil
	}
	if a.mcpMgr == nil {
		mgr, err := mcpsvc.NewManagerFromConfig(ctx, a.cfg.MCP.Servers, strict)
		if err != nil {
			return err
		}
		a.mcpMgr = mgr
	}
	return nil
}

// ensure MCP manager exists before building agent-mode registry.
func (a *app) ensureMCP(ctx context.Context, perm *permission.Engine, strict bool) error {
	if err := a.attachMCP(ctx, strict); err != nil {
		return err
	}
	if a.mcpMgr != nil {
		perm.SetWriteToolDetector(a.mcpMgr.IsWriteTool)
	}
	return nil
}
