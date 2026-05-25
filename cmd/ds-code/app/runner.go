package app

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/wzhejunqiu/ds-code/cmd/ds-code/slashcmd"
	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/audit"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm/deepseek"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	agenttool "github.com/wzhejunqiu/ds-code/internal/tool/builtin/agent"
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

	subStore := a.subStore
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
		Hooks:       agent.LoadHooks(a.Cfg.ProjectRoot),
	}
	// Wire async agent notification draining from the spawn service.
	if at, ok := bundle.reg.Get("agent"); ok {
		if agt, ok := at.(*agenttool.AgentTool); ok {
			svc := agt.SpawnService()
			svc.Hooks = runner.Hooks
			svc.ParentContext = ctxSvc
			runner.DrainNotifications = func(ctx context.Context) string {
				return formatNotifications(svc, spawn.PrioNow, spawn.PrioNext)
			}
			runner.DrainNotificationsLater = func(ctx context.Context, sessionID string) {
				notices := svc.DrainNotifications(spawn.PrioLater)
				for _, n := range notices {
					_ = store.AppendMessage(ctx, session.Message{
						SessionID: sessionID,
						Role:      role.User,
						Content:   n.FormatXML(),
					})
				}
			}
			svc.CleanupExpiredWorktrees(context.Background())
		}
	}
	return runner, store, ctxSvc, nil
}

func (a *App) spawnService(runner *agent.Runner) *spawn.Service {
	if runner == nil || runner.Tools == nil {
		return nil
	}
	at, ok := runner.Tools.Get("agent")
	if !ok {
		return nil
	}
	agt, ok := at.(*agenttool.AgentTool)
	if !ok {
		return nil
	}
	return agt.SpawnService()
}

func (a *App) cleanupSessionWorktrees(ctx context.Context, runner *agent.Runner, sessionID string) {
	if svc := a.spawnService(runner); svc != nil && sessionID != "" {
		svc.CleanupSessionWorktrees(ctx, sessionID)
	}
}

func (a *App) spawnRunner(runner *agent.Runner) slashcmd.SpawnRunner {
	return a.spawnService(runner)
}

func formatNotifications(svc *spawn.Service, prios ...spawn.NotificationPriority) string {
	var parts []string
	for _, p := range prios {
		for _, n := range svc.DrainNotifications(p) {
			parts = append(parts, n.FormatXML())
		}
	}
	return strings.Join(parts, "\n")
}
