package app

import (
	"context"
	"io"
	"os"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/audit"
	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm/deepseek"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
	"go.uber.org/zap"
)

func (a *App) newRunner(out io.Writer) (*agent.Runner, session.Store, *ctxpkg.Service, error) {
	logging.L().Info("building agent runner",
		zap.String("project_root", a.Cfg.ProjectRoot),
		zap.String("run_mode", a.Cfg.RunMode),
		zap.String("permission", a.Cfg.Permission.Mode),
	)
	store, err := a.openStore()
	if err != nil {
		return nil, nil, nil, err
	}
	interactive := permission.IsInteractiveTTY()
	perm := permission.NewEngine(a.Cfg.Permission.Mode, a.Cfg.ProjectRoot, interactive)
	if interactive && a.Cfg.Permission.Mode == "ask" {
		perm.Prompter = permission.StdinPrompter(os.Stderr)
	}

	strict := a.Cfg.LLM.StrictTools
	gi, _ := tool.LoadGitignore(a.Cfg.ProjectRoot)

	if err := a.ensureMCP(context.Background(), perm, strict); err != nil {
		return nil, nil, nil, err
	}

	llmClient := deepseek.NewClient(a.Cfg)
	bundle, err := a.buildTools(context.Background(), perm, gi, strict, llmClient, a.Cfg.RunMode)
	if err != nil {
		return nil, nil, nil, err
	}

	agentsMD, err := ctxpkg.LoadAgentsMD(a.Cfg.ProjectRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	rules, err := ctxpkg.LoadRules(a.Cfg.ProjectRoot)
	if err != nil {
		return nil, nil, nil, err
	}

	subStore, _ := a.openSubagentStore()
	ctxSvc := &ctxpkg.Service{
		Cfg:      a.Cfg,
		Store:    store,
		Subagent: subStore,
		Tools:    bundle.reg,
		LLM:      llmClient,
		AgentsMD: agentsMD,
		Rules:    rules,
		AtExpander: &ctxpkg.AtExpander{
			Cfg:       a.Cfg,
			Perm:      perm,
			Gitignore: gi,
		},
	}

	maxTurns := a.Cfg.Agent.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 25
	}

	var auditLog *audit.Logger
	if a.Cfg.Audit.Enabled {
		auditLog = audit.NewLogger(config.DefaultAuditLogPath(a.Cfg.ProjectRoot))
	}

	cpStore, err := a.openCheckpointStore()
	if err != nil {
		return nil, nil, nil, err
	}

	runner := &agent.Runner{
		LLM:         llmClient,
		Tools:       bundle.reg,
		Perm:        perm,
		Sessions:    store,
		Context:     ctxSvc,
		Cfg:         a.Cfg,
		MaxTurns:    maxTurns,
		Out:         out,
		Audit:       auditLog,
		Checkpoints: cpStore,
	}
	return runner, store, ctxSvc, nil
}
