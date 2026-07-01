// Package agent implements the multi-round LLM + tool agent loop (see README.md).
package agent

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/audit"
	"github.com/wzhejunqiu/ds-code/internal/checkpoint"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/mcp/resultstore"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/security/classifier"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// NotificationFunc is called at the start of RunTurn to drain async agent completion notices.
// It returns the notification text to inject as a system reminder before the user message.
type NotificationFunc func(ctx context.Context) string

// Runner executes the agent loop.
type Runner struct {
	LLM                     llm.Client
	Tools                   *tool.Registry
	Perm                    *permission.Engine
	Sessions                session.Store
	Context                 *ctxpkg.Service
	Cfg                     *config.Config
	MaxTurns                int
	Out                     io.Writer
	Audit                   *audit.Logger
	Checkpoints             *checkpoint.Store
	MCPResults              *resultstore.Store
	Hooks                   *HookManager
	DrainNotifications      NotificationFunc
	DrainNotificationsLater DrainNotificationsLaterFunc

	// ForSubagent marks a child-agent runner (uses llm.subagent.fallback_model on 5xx).
	ForSubagent bool

	sessionStarted map[string]bool

	readPathsMu       sync.Mutex
	readPaths         map[string]map[string]struct{} // sessionID → canonical abs
	readPathsHydrated map[string]bool
}

// TurnResult is the outcome of a user turn.
type TurnResult struct {
	FinalContent           string
	FinalReasoning         string
	FinalReasoningDuration time.Duration
	TurnDuration           time.Duration
	Usage                  llm.Usage
	SubRounds              int
}

func (r *Runner) executeTool(ctx context.Context, sessionID string, tc llm.ToolCall) string {
	ctx, end := trace.Start(ctx, trace.SpanTool(tc.Name),
		attribute.String(trace.AttrSessionID, sessionID),
		attribute.String(trace.AttrToolName, tc.Name),
		attribute.String(trace.AttrToolCallID, tc.ID),
	)
	defer end()
	if tool.NameAgent.Matches(tc.Name) && r.Cfg.Tools.Agent.ForkEnabled {
		ctx = r.enrichAgentForkContext(ctx, sessionID, tc)
	}
	rawArgs := []byte(tc.Arguments)
	if r.Hooks != nil {
		results := r.Hooks.Run(ctx, HookPreToolUse, hookInputForTool(sessionID, tc, rawArgs, nil))
		rawArgs = applyPreToolUseResults(tc.Name, rawArgs, results)
		tc.Arguments = string(rawArgs)
	}
	args := tool.ArgsMap(rawArgs)
	if tool.NameShell.Matches(tc.Name) && r.Audit != nil {
		if cmd, _ := args["command"].(string); cmd != "" {
			dec, reason := classifier.Classify(cmd)
			_ = r.Audit.LogWithReason(tc.Name, rawArgs, string(dec), reason)
		}
	}
	parentModel := ""
	if sess, err := r.Sessions.Get(ctx, sessionID); err == nil {
		parentModel = sess.Model
	}
	execCtx := WithToolInvocation(ctx, sessionID, tc.ID, parentModel)
	// web_fetch uses PrepareWebFetch (not Check) so AllowOnce approval propagates via execCtx.
	if tool.NameWebFetch.Matches(tc.Name) {
		var err error
		execCtx, err = r.Perm.PrepareWebFetch(execCtx, args)
		if err != nil {
			logging.L().Info("tool denied", zap.String("session_id", sessionID), zap.String("tool", tc.Name), zap.Error(err))
			return ctxpkg.FormatToolError(tc.Name, tc.ID, err)
		}
	} else if err := r.Perm.Check(tc.Name, args); err != nil {
		logging.L().Info("tool denied", zap.String("session_id", sessionID), zap.String("tool", tc.Name), zap.Error(err))
		return ctxpkg.FormatToolError(tc.Name, tc.ID, err)
	}
	if err := r.recordCheckpoint(ctx, sessionID, tc.Name, args); err != nil {
		logging.L().Debug("checkpoint failed",
			zap.String("session_id", sessionID),
			zap.String("tool", tc.Name),
			zap.Error(err),
		)
		return ctxpkg.FormatToolError(tc.Name, tc.ID, fmt.Errorf("checkpoint: %w", err))
	}
	if r.Audit != nil && !tool.NameShell.Matches(tc.Name) {
		_ = r.Audit.Log(tc.Name, rawArgs)
	}
	out, err := r.Tools.Execute(execCtx, tc.Name, rawArgs)
	if r.Hooks != nil {
		r.Hooks.Run(ctx, HookPostToolUse, hookInputForTool(sessionID, tc, rawArgs, err))
	}
	if err != nil {
		logging.L().Info("tool error", zap.String("session_id", sessionID), zap.String("tool", tc.Name), zap.Error(err))
		return ctxpkg.FormatToolError(tc.Name, tc.ID, err)
	}
	logging.L().Debug("tool ok", zap.String("session_id", sessionID), zap.String("tool", tc.Name), zap.Int("result_chars", len(out)))
	return ctxpkg.FormatToolResult(tc.Name, tc.ID, out)
}

func (r *Runner) enrichAgentForkContext(ctx context.Context, sessionID string, tc llm.ToolCall) context.Context {
	view, err := r.Context.BuildAPIContext(ctx, sessionID)
	if err != nil {
		return ctx
	}
	ctx = WithRenderedSystem(ctx, view.MergedSystem())

	var parentCalls []llm.ToolCall
	for i := len(view.Messages) - 1; i >= 0; i-- {
		m := view.Messages[i]
		if m.Role == role.Assistant && len(m.ToolCalls) > 0 {
			parentCalls = m.ToolCalls
			break
		}
	}
	if len(parentCalls) == 0 {
		parentCalls = []llm.ToolCall{tc}
	}
	return WithForkContext(ctx, ForkContext{
		ParentMessages:  view.Messages,
		ParentToolCalls: parentCalls,
	})
}
