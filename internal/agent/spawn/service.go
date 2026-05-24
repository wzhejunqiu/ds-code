package spawn

import (
	"context"
	"fmt"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
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
}

// NewService creates a spawn service wired to the parent session.
func NewService(cfg *config.Config, perm *permission.Engine, parentReg *tool.Registry, llmClient llm.Client, store subagentstore.Store) *Service {
	nq := NewNotificationQueue()
	return &Service{
		Registry:          NewRegistry(),
		Perm:              perm,
		ParentReg:         parentReg,
		LLM:               llmClient,
		Store:             store,
		Cfg:               cfg,
		BackgroundManager: NewBackgroundManager(nq),
		NotifyQueue:       nq,
	}
}

// Handle processes an agent tool call from start to finish.
func (s *Service) Handle(ctx context.Context, inv agent.ToolInvocation, params Params, interactive bool) (string, error) {
	decision, err := Route(params, inv, s.Registry, s.Cfg, interactive)
	if err != nil {
		return "", err
	}

	model := ResolveModel(decision.Model, decision.Definition.Model, s.Cfg)

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
		ThinkingType:     resolveThinkingType(s.Cfg, decision.Definition),
		Background:       decision.Background,
		OutputPath:       outputPath,
		IsolationMode:    decision.Isolation,
	})
	if err != nil {
		return "", err
	}

	parent := agent.TurnCallbacksFromContext(ctx)
	if parent != nil && parent.OnSubagentStart != nil {
		parent.OnSubagentStart(run.ID, decision.Description, decision.Prompt, decision.Definition.Type, run.Background)
	}

	// Fork recursive guard + message construction
	if decision.IsFork {
		fc, ok := agent.ForkContextFromContext(ctx)
		if !ok {
			return "", fmt.Errorf("fork: missing parent context")
		}
		if IsInForkChild(fc.ParentMessages) {
			return "", fmt.Errorf("fork: recursive fork detected — fork children cannot spawn fork sub-agents")
		}
		decision.Definition.Type = "fork"
	}

	if decision.Background {
		s.BackgroundManager.Start(ctx, s.Cfg, s.LLM, run, decision.Definition, s.Perm, s.ParentReg, s.Store, parent, outputPath)
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
		summary, runErr := ExecuteRun(ctx, s.Cfg, s.LLM, run, decision.Definition, s.Perm, s.ParentReg, s.Store, cb, 0)
		return s.finishSync(ctx, run, decision, parent, outputPath, summary, runErr)
	}

	if outputPath == "" {
		outputPath = agentOutputPath(s.Cfg.ProjectDataDir, inv.SessionID, inv.ToolCallID)
	}

	done := make(chan executeResult, 1)
	runCtx, cancel := context.WithCancel(ctx)
	go func() {
		summary, err := ExecuteRun(runCtx, s.Cfg, s.LLM, run, decision.Definition, s.Perm, s.ParentReg, s.Store, cb, 0)
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
		AgentID:    run.ID,
		ToolUseID:  run.ParentToolCallID,
		OutputFile: outputPath,
		Status:     statusStr,
		Summary:    summaryText,
		Result:     summary,
		DurationMS: durationMS,
	}, PrioNext)
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

