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
// It wires routing, persistence, sync/async execution, and result delivery.
type Service struct {
	// Registry resolves subagent_type to built-in agent definitions.
	Registry *Registry
	// Perm is the shared permission engine for sub-agents.
	Perm *permission.Engine
	// ParentReg is the parent session tool registry used to build the sub-agent tool pool.
	ParentReg *tool.Registry
	// LLM is the client used for sub-agent chat turns.
	LLM llm.Client
	// Store persists sub-agent run records and messages.
	Store subagentstore.Store
	// Cfg holds project and user configuration.
	Cfg *config.Config
	// BackgroundManager tracks in-flight async agents and promoted sync runs.
	BackgroundManager *BackgroundManager
	// NotifyQueue holds completion notices drained into the parent conversation.
	NotifyQueue *NotificationQueue
	// Hooks runs SubagentStart/Stop lifecycle hooks; injected during app assembly.
	Hooks *agent.HookManager
	// ParentContext reads the parent session API view and thinking type for fork spawns.
	ParentContext *ctxpkg.Service
	// Worktrees creates isolated git worktrees when isolation=worktree.
	Worktrees *worktree.Manager
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
		BackgroundManager: NewBackgroundManager(nq), // shares queue with Service.NotifyQueue
		NotifyQueue:       nq,
		Worktrees:         worktree.NewManager(wtBase),
	}
}

// Handle processes an agent tool call end to end: route, create run, execute
// synchronously or in the background, and return the tool result payload.
func (s *Service) Handle(ctx context.Context, inv agent.ToolInvocation, params Params, interactive bool) (string, error) {
	// Fork / Sync / Async based on params, agent type, and config.
	decision, err := Route(ctx, params, inv, s.Registry, s.Cfg, interactive)
	if err != nil {
		return "", err
	}

	// Caller override → type default → global subagent model.
	model := ResolveModel(decision.Model, decision.Definition.Model, s.Cfg)

	if decision.Isolation == "worktree" {
		if decision.Definition.Type != AgentTypeGeneralPurpose {
			return "", fmt.Errorf("isolation worktree is only supported for general-purpose agents")
		}
	}

	// Optional git worktree: create before persisting so CreateRun failure can roll back.
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

	run, err := s.Store.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID:  inv.SessionID,
		ParentToolCallID: inv.ToolCallID,
		AgentType:        decision.Definition.Type.String(),
		SpawnKind:        decision.SpawnKind,
		Label:            truncateLabel(decision.Description, decision.Prompt, 48),
		Prompt:           decision.Prompt,
		Model:            model,
		ReasoningEffort:  s.Cfg.LLM.ResolveSubagentReasoningEffort(),
		ThinkingType:     resolveThinkingType(s.Cfg, decision.Definition, s.parentSessionThinking(ctx, inv.SessionID), decision.IsFork),
		Background:       decision.Background,
		OutputPath:       "",
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
			AgentType: decision.Definition.Type.String(),
		}))
	}
	if parent != nil && parent.OnSubagentStart != nil {
		parent.OnSubagentStart(run.ID, decision.Description, decision.Prompt, decision.Definition.Type.String(), run.Background)
	}

	// Fork requires parent API messages in context; override type for downstream overlays.
	if decision.IsFork {
		if _, ok := agent.ForkContextFromContext(ctx); !ok {
			return "", fmt.Errorf("fork: missing parent context")
		}
		decision.Definition.Type = AgentTypeFork
	}

	// Async path: detach goroutine and return launch JSON immediately.
	if decision.Background {
		s.BackgroundManager.Start(ctx, s.Cfg, s.LLM, run, decision.Definition, s.Perm, s.ParentReg, s.Store, parent, s.Hooks, s.cleanupWorktreeImmediate)
		return fmt.Sprintf(`{"status":"async_launched","agent_id":"%s","description":"%s"}`, run.ID, decision.Description), nil
	}

	return s.runSync(ctx, inv, run, decision, parent)
}

// executeResult carries the outcome of a goroutine-started agent run.
type executeResult struct {
	summary string
	err     error
}

// runSync executes an agent run in the foreground. When auto_background_after is
// configured, it may promote a long-running sync run to async and return early.
func (s *Service) runSync(ctx context.Context, inv agent.ToolInvocation, run subagentstore.Run, decision RouteDecision, parent *agent.TurnCallbacks) (string, error) {
	cb := agent.SubagentToolCallbacks(parent, run.ID)
	timeoutSec := s.Cfg.Tools.Agent.AutoBackgroundAfter
	if timeoutSec <= 0 {
		// No promote threshold: block until ExecuteRun completes.
		summary, runErr := ExecuteRun(ctx, s.Cfg, s.LLM, run, decision.Definition, s.Perm, s.ParentReg, s.Store, cb, s.Hooks, 0)
		return s.finishSync(ctx, run, decision, parent, summary, runErr)
	}

	// Run in goroutine so we can race completion against auto_background_after.
	done := make(chan executeResult, 1)
	runCtx, cancel := context.WithCancel(ctx)
	promoted := false
	defer func() {
		// Cancel the goroutine unless we handed ownership to waitPromoted.
		if !promoted {
			cancel()
		}
	}()
	go func() {
		summary, err := ExecuteRun(runCtx, s.Cfg, s.LLM, run, decision.Definition, s.Perm, s.ParentReg, s.Store, cb, s.Hooks, 0)
		done <- executeResult{summary: summary, err: err}
	}()

	select {
	case res := <-done:
		return s.finishSync(ctx, run, decision, parent, res.summary, res.err)
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		// Promote to async: parent tool returns immediately; result arrives via NotifyQueue.
		promoted = true
		_ = s.Store.SetRunBackground(ctx, run.ID, true)
		run.Background = true
		s.BackgroundManager.RegisterPromoted(run.ID, decision.Definition.Type, run.Label)
		go s.waitPromoted(ctx, run, decision, parent, done, cancel)
		return fmt.Sprintf(`{"status":"async_launched","agent_id":"%s","description":"%s"}`, run.ID, decision.Description), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// waitPromoted blocks until a promoted sync run finishes and delivers the result asynchronously.
func (s *Service) waitPromoted(ctx context.Context, run subagentstore.Run, decision RouteDecision, parent *agent.TurnCallbacks, done <-chan executeResult, cancel context.CancelFunc) {
	defer cancel()
	defer s.BackgroundManager.CompletePromoted(run.ID)
	res := <-done
	// Promoted runs use the async finish path (NotifyQueue, not inline tool return).
	s.finishAsync(ctx, run, decision, parent, res.summary, res.err)
}

// finishSync persists run status, fires hooks, and returns the inline tool result for sync execution.
func (s *Service) finishSync(ctx context.Context, run subagentstore.Run, decision RouteDecision, parent *agent.TurnCallbacks, summary string, runErr error) (string, error) {
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
		in := agent.HookInput{AgentID: run.ID, AgentType: decision.Definition.Type.String()}
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
	// Inline summary or spill to output file based on size limits.
	resultStatus, err := resultStatusFromStore(status)
	if err != nil {
		return "", err
	}
	delivered := DeliverResult(s.Cfg.ProjectDataDir, run.ParentSessionID, run.ParentToolCallID, summary, resultStatus, nil, s.Cfg)
	return formatSyncToolReturn(decision.Description, delivered), nil
}

// finishAsync persists run status for background or promoted runs and enqueues a completion notification.
func (s *Service) finishAsync(ctx context.Context, run subagentstore.Run, decision RouteDecision, parent *agent.TurnCallbacks, summary string, runErr error) {
	startTime := run.CreatedAt
	status := subagentstore.StatusCompleted
	errMsg := ""
	if runErr != nil {
		// Context cancellation means parent turn was aborted, not an agent logic error.
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
		in := agent.HookInput{AgentID: run.ID, AgentType: decision.Definition.Type.String()}
		if runErr != nil {
			in.Error = runErr.Error()
		}
		s.Hooks.Run(ctx, agent.HookSubagentStop, agent.MarshalHookInput(in))
	}

	resultStatus, err := resultStatusFromStore(status)
	if err != nil {
		logging.L().Error("map subagent result status", zap.String("run_id", run.ID), zap.Error(err))
		return
	}
	summaryText := agentSummaryText(run.Label, resultStatus, runErr)
	delivered := DeliverResult(s.Cfg.ProjectDataDir, run.ParentSessionID, run.ParentToolCallID, summary, resultStatus, runErr, s.Cfg)
	n := Notification{
		AgentID:        run.ID,
		ToolUseID:      run.ParentToolCallID,
		Status:         resultStatus,
		Summary:        summaryText,
		DurationMS:     durationMS,
		ToolUseCount:   countToolUses(ctx, s.Store, run.ID),
		WorktreePath:   run.WorktreePath,
		WorktreeBranch: run.WorktreeBranch,
	}
	if delivered.Inline {
		n.Result = delivered.Body
	} else {
		n.OutputFile = delivered.OutputPath
	}
	// Parent Runner drains this queue and injects XML into the next turn.
	s.NotifyQueue.Enqueue(n, notificationPriority(ctx))
	if parent != nil && parent.OnSubagentEnd != nil {
		parent.OnSubagentEnd(run.ID, summary, runErr)
	}
	logging.L().Info("promoted agent finished",
		zap.String("run_id", run.ID),
		zap.String("status", resultStatus.String()),
	)
}

// agentSummaryText builds a short human-readable summary for completion notifications.
func agentSummaryText(label string, status ResultStatus, runErr error) string {
	if runErr != nil {
		return fmt.Sprintf("Agent %q failed: %s", label, runErr.Error())
	}
	return fmt.Sprintf("Agent %q %s", label, status)
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

// truncateLabel prefers description over prompt when building a run label.
func truncateLabel(desc, prompt string, max int) string {
	if desc != "" {
		return truncate(desc, max)
	}
	return truncate(prompt, max)
}

// truncate shortens s to max bytes and appends "..." when truncated.
func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// parentSessionThinking returns the parent session thinking type for fork inheritance.
func (s *Service) parentSessionThinking(ctx context.Context, sessionID string) string {
	if s.ParentContext == nil || s.ParentContext.Store == nil || sessionID == "" {
		return ""
	}
	sess, err := s.ParentContext.Store.Get(ctx, sessionID)
	if err != nil {
		return ""
	}
	// Fork spawns inherit the parent session's thinking mode when not overridden.
	return sess.ThinkingType
}

// worktreeSlug sanitizes a tool call ID into a safe worktree directory name.
func worktreeSlug(toolCallID string) string {
	// Replace unsafe characters so the slug is a valid directory name.
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

// worktreeOpts builds worktree creation options from config with package defaults.
func worktreeOpts(cfg *config.Config) worktree.CreateOptions {
	if cfg == nil {
		return worktree.CreateOptions{SparsePaths: []string{"/*"}, SymlinkDirs: []string{"node_modules", ".venv", "vendor"}}
	}
	opts := worktree.CreateOptions{
		SparsePaths: cfg.Tools.Agent.WorktreeSparsePaths,
		SymlinkDirs: cfg.Tools.Agent.WorktreeSymlinkDirs,
	}
	// Fall back to sparse checkout of entire tree and common dependency symlinks.
	if len(opts.SparsePaths) == 0 {
		opts.SparsePaths = []string{"/*"}
	}
	if len(opts.SymlinkDirs) == 0 {
		opts.SymlinkDirs = []string{"node_modules", ".venv", "vendor"}
	}
	return opts
}
