package spawn

import (
	"context"
	"fmt"
	"time"

	"path/filepath"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/worktree"
	"go.uber.org/zap"
)

// Service is the single entry point for all agent spawns.
type Service struct {
	Registry          *Registry
	Perm              *permission.Engine
	ParentReg         *tool.Registry
	LLM               llm.Client
	Store             subagentstore.Store
	Cfg               *config.Config
	BackgroundManager *BackgroundManager
	NotifyQueue       *NotificationQueue
	Hooks             *agent.HookManager
	ParentContext     *ctxpkg.Service
	Worktrees         *worktree.Manager
}

// NewService creates a spawn service wired to the parent session.
func NewService(cfg *config.Config, perm *permission.Engine, parentReg *tool.Registry, llmClient llm.Client, store subagentstore.Store) *Service {
	nq := NewNotificationQueue()
	wtBase := filepath.Join(cfg.ProjectDataDir, "worktrees")
	return &Service{
		Registry:          NewRegistry(),
		Perm:              perm,
		ParentReg:         parentReg,
		LLM:               llmClient,
		Store:             store,
		Cfg:               cfg,
		BackgroundManager: NewBackgroundManager(nq),
		NotifyQueue:       nq,
		Worktrees:         worktree.NewManager(wtBase),
	}
}

// Handle processes an agent tool call from start to finish.
func (s *Service) Handle(ctx context.Context, inv agent.ToolInvocation, params Params, interactive bool) (string, error) {
	decision, err := Route(ctx, params, inv, s.Registry, s.Cfg, interactive)
	if err != nil {
		return "", err
	}

	model := ResolveModel(decision.Model, decision.Definition.Model, s.Cfg)

	if decision.Isolation == "worktree" {
		if decision.Definition.Type != "general-purpose" {
			return "", fmt.Errorf("isolation worktree is only supported for general-purpose agents")
		}
	}

	wtPath, wtBranch := "", ""
	if decision.Isolation == "worktree" && s.Worktrees != nil {
		slug := worktreeSlug(inv.ToolCallID)
		var err error
		wtPath, wtBranch, err = s.Worktrees.Create(ctx, s.Cfg.ProjectRoot, slug, worktreeOpts(s.Cfg))
		if err != nil {
			return "", err
		}
		if err := s.Worktrees.ValidatePath(wtPath); err != nil {
			s.cleanupWorktreeImmediate(ctx, subagentstore.Run{WorktreePath: wtPath, WorktreeBranch: wtBranch})
			return "", fmt.Errorf("worktree path: %w", err)
		}
	}

	outputPath := ""
	if decision.Background {
		outputPath = agentOutputPath(s.Cfg.ProjectDataDir, inv.SessionID, inv.ToolCallID)
	}

	run, err := s.Store.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID:  inv.SessionID,
		ParentToolCallID: inv.ToolCallID,
		AgentType:        decision.Definition.Type,
		SpawnKind:        decision.SpawnKind,
		Label:            truncateLabel(decision.Description, decision.Prompt, 48),
		Prompt:           decision.Prompt,
		Model:            model,
		ReasoningEffort:  s.Cfg.LLM.ResolveSubagentReasoningEffort(),
		ThinkingType:     resolveThinkingType(s.Cfg, decision.Definition, s.parentSessionThinking(ctx, inv.SessionID), decision.IsFork),
		Background:       decision.Background,
		OutputPath:       outputPath,
		IsolationMode:    decision.Isolation,
		WorktreePath:     wtPath,
		WorktreeBranch:   wtBranch,
	})
	if err != nil {
		if wtPath != "" {
			s.cleanupWorktreeImmediate(ctx, subagentstore.Run{WorktreePath: wtPath, WorktreeBranch: wtBranch})
		}
		return "", err
	}
	run.WorktreePath = wtPath
	run.WorktreeBranch = wtBranch

	parent := agent.TurnCallbacksFromContext(ctx)
	if s.Hooks != nil {
		s.Hooks.Run(ctx, agent.HookSubagentStart, agent.MarshalHookInput(agent.HookInput{
			SessionID: inv.SessionID,
			AgentID:   run.ID,
			AgentType: decision.Definition.Type,
		}))
	}
	if parent != nil && parent.OnSubagentStart != nil {
		parent.OnSubagentStart(run.ID, decision.Description, decision.Prompt, decision.Definition.Type, run.Background)
	}

	// Fork recursive guard + message construction
	if decision.IsFork {
		if _, ok := agent.ForkContextFromContext(ctx); !ok {
			return "", fmt.Errorf("fork: missing parent context")
		}
		decision.Definition.Type = "fork"
	}

	if decision.Background {
		s.BackgroundManager.Start(ctx, s.Cfg, s.LLM, run, decision.Definition, s.Perm, s.ParentReg, s.Store, parent, outputPath, s.Hooks, s.cleanupWorktreeImmediate)
		return fmt.Sprintf(`{"status":"async_launched","agent_id":"%s","description":"%s","output_file":"%s"}`, run.ID, decision.Description, outputPath), nil
	}

	return s.runSync(ctx, inv, run, decision, parent, outputPath)
}

type executeResult struct {
	summary string
	err     error
}

func (s *Service) runSync(ctx context.Context, inv agent.ToolInvocation, run subagentstore.Run, decision RouteDecision, parent *agent.TurnCallbacks, outputPath string) (string, error) {
	cb := agent.SubagentToolCallbacks(parent, run.ID)
	timeoutSec := s.Cfg.Tools.Agent.AutoBackgroundAfter
	if timeoutSec <= 0 {
		summary, runErr := ExecuteRun(ctx, s.Cfg, s.LLM, run, decision.Definition, s.Perm, s.ParentReg, s.Store, cb, s.Hooks, 0)
		return s.finishSync(ctx, run, decision, parent, outputPath, summary, runErr)
	}

	if outputPath == "" {
		outputPath = agentOutputPath(s.Cfg.ProjectDataDir, inv.SessionID, inv.ToolCallID)
	}

	done := make(chan executeResult, 1)
	runCtx, cancel := context.WithCancel(ctx)
	go func() {
		summary, err := ExecuteRun(runCtx, s.Cfg, s.LLM, run, decision.Definition, s.Perm, s.ParentReg, s.Store, cb, s.Hooks, 0)
		done <- executeResult{summary: summary, err: err}
	}()

	select {
	case res := <-done:
		return s.finishSync(ctx, run, decision, parent, outputPath, res.summary, res.err)
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		_ = s.Store.SetRunBackground(ctx, run.ID, true)
		run.Background = true
		s.BackgroundManager.RegisterPromoted(run.ID, decision.Definition.Type, run.Label)
		go s.waitPromoted(ctx, run, decision, parent, outputPath, done)
		return fmt.Sprintf(`{"status":"async_launched","agent_id":"%s","description":"%s","output_file":"%s"}`, run.ID, decision.Description, outputPath), nil
	case <-ctx.Done():
		cancel()
		return "", ctx.Err()
	}
}

func (s *Service) waitPromoted(ctx context.Context, run subagentstore.Run, decision RouteDecision, parent *agent.TurnCallbacks, outputPath string, done <-chan executeResult) {
	defer s.BackgroundManager.CompletePromoted(run.ID)
	res := <-done
	s.finishAsync(ctx, run, decision, parent, outputPath, res.summary, res.err)
}

func (s *Service) finishSync(ctx context.Context, run subagentstore.Run, decision RouteDecision, parent *agent.TurnCallbacks, outputPath string, summary string, runErr error) (string, error) {
	status := subagentstore.StatusCompleted
	errMsg := ""
	if runErr != nil {
		status = subagentstore.StatusError
		errMsg = runErr.Error()
	}
	_ = s.Store.FinishRun(ctx, run.ID, status, errMsg)
	if runErr != nil {
		s.cleanupWorktreeImmediate(ctx, run)
	}
	if s.Hooks != nil {
		in := agent.HookInput{AgentID: run.ID, AgentType: decision.Definition.Type}
		if runErr != nil {
			in.Error = runErr.Error()
		}
		s.Hooks.Run(ctx, agent.HookSubagentStop, agent.MarshalHookInput(in))
	}
	if parent != nil && parent.OnSubagentEnd != nil {
		parent.OnSubagentEnd(run.ID, summary, runErr)
	}
	if runErr != nil {
		return "", runErr
	}
	if decision.Description != "" {
		return fmt.Sprintf("[%s]\n%s", decision.Description, summary), nil
	}
	return summary, nil
}

func (s *Service) finishAsync(ctx context.Context, run subagentstore.Run, decision RouteDecision, parent *agent.TurnCallbacks, outputPath, summary string, runErr error) {
	startTime := run.CreatedAt
	status := subagentstore.StatusCompleted
	errMsg := ""
	if runErr != nil {
		if ctx.Err() != nil {
			status = subagentstore.StatusKilled
		} else {
			status = subagentstore.StatusError
		}
		errMsg = runErr.Error()
	}
	durationMS := time.Since(startTime).Milliseconds()
	_ = s.Store.FinishRun(ctx, run.ID, status, errMsg)
	if runErr != nil || status == subagentstore.StatusKilled {
		s.cleanupWorktreeImmediate(ctx, run)
	}
	if s.Hooks != nil {
		in := agent.HookInput{AgentID: run.ID, AgentType: decision.Definition.Type}
		if runErr != nil {
			in.Error = runErr.Error()
		}
		s.Hooks.Run(ctx, agent.HookSubagentStop, agent.MarshalHookInput(in))
	}

	statusStr := "completed"
	switch status {
	case subagentstore.StatusError:
		statusStr = "failed"
	case subagentstore.StatusKilled:
		statusStr = "killed"
	}
	summaryText := fmt.Sprintf("Agent %q %s", run.Label, statusStr)
	if runErr != nil {
		summaryText = fmt.Sprintf("Agent %q failed: %s", run.Label, runErr.Error())
	}
	s.NotifyQueue.Enqueue(Notification{
		AgentID:        run.ID,
		ToolUseID:      run.ParentToolCallID,
		OutputFile:     outputPath,
		Status:         statusStr,
		Summary:        summaryText,
		Result:         summary,
		DurationMS:     durationMS,
		ToolUseCount:   countToolUses(ctx, s.Store, run.ID),
		WorktreePath:   run.WorktreePath,
		WorktreeBranch: run.WorktreeBranch,
	}, notificationPriority(ctx))
	if outputPath != "" {
		writeOutputFile(outputPath, summary, statusStr, runErr)
	}
	if parent != nil && parent.OnSubagentEnd != nil {
		parent.OnSubagentEnd(run.ID, summary, runErr)
	}
	logging.L().Info("promoted agent finished",
		zap.String("run_id", run.ID),
		zap.String("status", statusStr),
	)
}

// DrainNotifications returns pending completion notices for injection into the main conversation.
func (s *Service) DrainNotifications(prio NotificationPriority) []Notification {
	return s.NotifyQueue.Drain(prio)
}

// HasPendingNotifications reports whether any notifications are queued.
func (s *Service) HasPendingNotifications() bool {
	return s.NotifyQueue.HasPending()
}

// agentOutputPath builds the path for async agent output files.
func agentOutputPath(dataDir, parentSessionID, toolCallID string) string {
	return fmt.Sprintf("%s/agents/%s/%s.output", dataDir, parentSessionID, toolCallID)
}

func truncateLabel(desc, prompt string, max int) string {
	if desc != "" {
		return truncate(desc, max)
	}
	return truncate(prompt, max)
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func (s *Service) parentSessionThinking(ctx context.Context, sessionID string) string {
	if s.ParentContext == nil || s.ParentContext.Store == nil || sessionID == "" {
		return ""
	}
	sess, err := s.ParentContext.Store.Get(ctx, sessionID)
	if err != nil {
		return ""
	}
	return sess.ThinkingType
}

func worktreeSlug(toolCallID string) string {
	slug := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, toolCallID)
	if len(slug) > 32 {
		slug = slug[:32]
	}
	if slug == "" {
		slug = "agent"
	}
	return slug
}

func worktreeOpts(cfg *config.Config) worktree.CreateOptions {
	if cfg == nil {
		return worktree.CreateOptions{SparsePaths: []string{"/*"}, SymlinkDirs: []string{"node_modules", ".venv", "vendor"}}
	}
	opts := worktree.CreateOptions{
		SparsePaths: cfg.Tools.Agent.WorktreeSparsePaths,
		SymlinkDirs: cfg.Tools.Agent.WorktreeSymlinkDirs,
	}
	if len(opts.SparsePaths) == 0 {
		opts.SparsePaths = []string{"/*"}
	}
	if len(opts.SymlinkDirs) == 0 {
		opts.SymlinkDirs = []string{"node_modules", ".venv", "vendor"}
	}
	return opts
}
