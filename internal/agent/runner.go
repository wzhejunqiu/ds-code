// Package agent implements the multi-round LLM + tool agent loop (see README.md).
package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/hejunqiu/ds-code/internal/audit"
	"github.com/hejunqiu/ds-code/internal/checkpoint"
	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
	"go.uber.org/zap"
)

// Runner executes the agent loop.
type Runner struct {
	LLM     llm.Client
	Tools   *tool.Registry
	Perm    *permission.Engine
	Sessions session.Store
	Context *ctxpkg.Service
	Cfg      *config.Config
	MaxTurns int
	Out          io.Writer
	Audit        *audit.Logger
	Checkpoints  *checkpoint.Store
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

// RunTurn handles one user message through sub-rounds until no tool_calls or max turns.
// Optional cb streams deltas and tool events to the TUI; nil cb writes final text to r.Out only.
func (r *Runner) RunTurn(ctx context.Context, sessionID, userText string, cb *TurnCallbacks) (*TurnResult, error) {
	if cb != nil {
		ctx = WithTurnCallbacks(ctx, cb)
	}
	logging.L().Info("user turn start", zap.String("session_id", sessionID), zap.Int("chars", len(userText)))
	expanded, err := r.Context.ExpandUserText(userText)
	if err != nil {
		return nil, fmt.Errorf("expand @ references: %w", err)
	}
	if err := r.Sessions.AppendMessage(ctx, session.Message{
		SessionID: sessionID,
		Role:      role.User,
		Content:   expanded,
	}); err != nil {
		return nil, err
	}

	sess, err := r.Sessions.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	r.Context.BeginUserTurn()
	defer r.Context.EndUserTurn()

	turnStart := time.Now()
	result := &TurnResult{}
	// Each iteration is one LLM request; tool results feed the next iteration.
	for round := 0; round < r.MaxTurns; round++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// Close the prior assistant segment in the UI before tools / next model reply.
		if round > 0 && cb != nil && cb.OnAssistantSegmentEnd != nil {
			cb.OnAssistantSegmentEnd()
		}
		// Show planning label until the model stream produces content or reasoning.
		if cb != nil && cb.OnPlanningStart != nil {
			cb.OnPlanningStart()
		}

		logging.L().Debug("agent sub-round", zap.String("session_id", sessionID), zap.Int("round", round+1))
		view, maxTokens, err := r.Context.PrepareRequest(ctx, sessionID)
		if err != nil {
			return nil, err
		}

		req := llm.Request{
			MergedSystem:    view.MergedSystem(),
			Messages:        view.Messages,
			Model:           sess.Model,
			Tools:           r.Tools.Definitions(),
			MaxTokens:       maxTokens,
			Stream:          true,
			ThinkingType:    sess.ThinkingType,
			ReasoningEffort: sess.ReasoningEffort,
			UserID:          cacheScope(sessionID),
			StrictTools:     r.Cfg.LLM.StrictTools,
		}
		stream := &subRoundStream{}
		req.OnStream = r.attachStreamHandlers(cb, round, stream)

		logging.L().Info("LLM request",
			zap.String("session_id", sessionID),
			zap.Int("round", round+1),
			zap.String("model", req.Model),
			zap.Int("messages", len(req.Messages)),
			zap.Int("tools", len(req.Tools)),
		)
		resp, err := r.chatWithCompactRetry(ctx, sessionID, req)
		if err != nil {
			if !stream.planningDone && cb != nil && cb.OnPlanningEnd != nil {
				cb.OnPlanningEnd()
			}
			logging.L().Error("LLM request failed", zap.String("session_id", sessionID), zap.Error(err))
			return nil, err
		}
		if !stream.planningDone && cb != nil && cb.OnPlanningEnd != nil {
			cb.OnPlanningEnd()
		}
		logging.L().Debug("LLM response",
			zap.String("session_id", sessionID),
			zap.Int("tool_calls", len(resp.ToolCalls)),
			zap.Int("content_chars", len(resp.Content)),
		)

		_ = r.Sessions.AddUsage(ctx, sessionID, resp.Usage)
		result.Usage = resp.Usage
		result.SubRounds = round + 1

		if len(resp.ToolCalls) == 0 {
			return r.finishTerminalRound(ctx, sessionID, resp, stream, turnStart, result, cb)
		}
		if err := r.appendAssistantWithTools(ctx, sessionID, resp, stream); err != nil {
			return nil, err
		}
		if err := r.runToolCalls(ctx, sessionID, resp.ToolCalls, resp, stream, cb); err != nil {
			return nil, err
		}
	}
	logging.L().Warn("exceeded max sub-rounds", zap.String("session_id", sessionID), zap.Int("max", r.MaxTurns))
	return nil, fmt.Errorf("agent: exceeded max sub-rounds (%d)", r.MaxTurns)
}

func (r *Runner) executeTool(ctx context.Context, sessionID string, tc llm.ToolCall) string {
	rawArgs := []byte(tc.Arguments)
	args := tool.ArgsMap(rawArgs)
	if err := r.Perm.Check(tc.Name, args); err != nil {
		logging.L().Info("tool denied", zap.String("session_id", sessionID), zap.String("tool", tc.Name), zap.Error(err))
		return ctxpkg.FormatToolError(tc.Name, tc.ID, err)
	}
	if err := r.recordCheckpoint(ctx, sessionID, tc.Name, args); err != nil {
		return ctxpkg.FormatToolError(tc.Name, tc.ID, fmt.Errorf("checkpoint: %w", err))
	}
	if r.Audit != nil {
		_ = r.Audit.Log(tc.Name, rawArgs)
	}
	out, err := r.Tools.Execute(WithToolInvocation(ctx, sessionID, tc.ID), tc.Name, rawArgs)
	if err != nil {
		logging.L().Info("tool error", zap.String("session_id", sessionID), zap.String("tool", tc.Name), zap.Error(err))
		return ctxpkg.FormatToolError(tc.Name, tc.ID, err)
	}
	logging.L().Debug("tool ok", zap.String("session_id", sessionID), zap.String("tool", tc.Name), zap.Int("result_chars", len(out)))
	return ctxpkg.FormatToolResult(tc.Name, tc.ID, out)
}

func cacheScope(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return hex.EncodeToString(sum[:])
}
