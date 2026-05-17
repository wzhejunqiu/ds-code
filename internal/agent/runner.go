package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hejunqiu/ds-code/internal/audit"
	"github.com/hejunqiu/ds-code/internal/checkpoint"
	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/deepseek"
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
func (r *Runner) RunTurn(ctx context.Context, sessionID, userText string, cb *TurnCallbacks) (*TurnResult, error) {
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
	for round := 0; round < r.MaxTurns; round++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if round > 0 && cb != nil && cb.OnAssistantSegmentEnd != nil {
			cb.OnAssistantSegmentEnd()
		}
		if round > 0 && cb != nil && cb.OnPlanningStart != nil {
			cb.OnPlanningStart()
		}

		logging.L().Debug("agent sub-round", zap.String("session_id", sessionID), zap.Int("round", round+1))
		view, maxTokens, err := r.Context.PrepareRequest(ctx, sessionID)
		if err != nil {
			return nil, err
		}

		toolDefs := r.Tools.Definitions()
		req := llm.Request{
			MergedSystem:    view.MergedSystem(),
			Messages:        view.Messages,
			Model:           sess.Model,
			Tools:           toolDefs,
			MaxTokens:       maxTokens,
			Stream:          true,
			ThinkingType:    sess.ThinkingType,
			ReasoningEffort: sess.ReasoningEffort,
			UserID:          cacheScope(sessionID),
			StrictTools:     r.Cfg.LLM.StrictTools,
		}
		var st streamTiming
		var roundContent strings.Builder
		streamedContent := false
		planningDone := round == 0
		endPlanning := func() {
			if planningDone || cb == nil || cb.OnPlanningEnd == nil {
				return
			}
			planningDone = true
			cb.OnPlanningEnd()
		}
		if cb != nil {
			req.OnStream = func(d llm.StreamDelta) {
				st.observe(d)
				if d.Content != "" || d.Reasoning != "" {
					endPlanning()
				}
				if d.Content != "" {
					roundContent.WriteString(d.Content)
					if cb.OnContentDelta != nil {
						streamedContent = true
						cb.OnContentDelta(d.Content)
					}
				}
				if d.Reasoning != "" && cb.OnReasoningDelta != nil {
					cb.OnReasoningDelta(d.Reasoning)
				}
			}
		}

		logging.L().Info("LLM request",
			zap.String("session_id", sessionID),
			zap.Int("round", round+1),
			zap.String("model", req.Model),
			zap.Int("messages", len(req.Messages)),
			zap.Int("tools", len(req.Tools)),
		)
		resp, err := r.LLM.Chat(ctx, req)
		if err != nil && deepseek.IsContextTooLong(err) {
			logging.L().Info("context too long, compacting", zap.String("session_id", sessionID))
			if compactErr := r.Context.CompactAPIContext(ctx, sessionID); compactErr != nil {
				return nil, fmt.Errorf("context too long; compact failed: %w", compactErr)
			}
			view, maxTokens, prepErr := r.Context.PrepareRequest(ctx, sessionID)
			if prepErr != nil {
				return nil, prepErr
			}
			req.Messages = view.Messages
			req.MergedSystem = view.MergedSystem()
			req.MaxTokens = maxTokens
			resp, err = r.LLM.Chat(ctx, req)
		}
		if err != nil {
			endPlanning()
			logging.L().Error("LLM request failed", zap.String("session_id", sessionID), zap.Error(err))
			return nil, err
		}
		endPlanning()
		logging.L().Debug("LLM response",
			zap.String("session_id", sessionID),
			zap.Int("tool_calls", len(resp.ToolCalls)),
			zap.Int("content_chars", len(resp.Content)),
		)

		_ = r.Sessions.AddUsage(ctx, sessionID, resp.Usage)
		result.Usage = resp.Usage
		result.SubRounds = round + 1

		tcJSON, _ := json.Marshal(resp.ToolCalls)
		reasoningDur := st.duration()
		assistantMsg := session.Message{
			SessionID:           sessionID,
			Role:                role.Assistant,
			Content:             resp.Content,
			ReasoningContent:    resp.ReasoningContent,
			ReasoningDurationMS: durationMS(reasoningDur),
			ToolCallsJSON:       string(tcJSON),
		}
		if len(resp.ToolCalls) == 0 {
			turnDur := time.Since(turnStart)
			assistantMsg.TurnDurationMS = durationMS(turnDur)
			result.TurnDuration = turnDur
			result.FinalReasoningDuration = reasoningDur
		}
		if err := r.Sessions.AppendMessage(ctx, assistantMsg); err != nil {
			return nil, err
		}

		if len(resp.ToolCalls) == 0 {
			result.FinalContent = resp.Content
			result.FinalReasoning = resp.ReasoningContent
			if cb != nil && cb.OnContentDelta != nil && !streamedContent {
				content := resp.Content
				if roundContent.Len() > 0 {
					content = roundContent.String()
				}
				if content != "" {
					cb.OnContentDelta(content)
				}
			}
			logging.L().Info("user turn done (no tools)",
				zap.String("session_id", sessionID),
				zap.Int("rounds", round+1),
			)
			if cb == nil && r.Out != nil && resp.Content != "" {
				_, _ = io.WriteString(r.Out, resp.Content)
			}
			return result, nil
		}

		roundText := resp.Content
		if roundContent.Len() > 0 {
			roundText = roundContent.String()
		}

		for _, tc := range resp.ToolCalls {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			logging.L().Info("tool execute",
				zap.String("session_id", sessionID),
				zap.String("tool", tc.Name),
				zap.String("call_id", tc.ID),
			)
			rawArgs := []byte(tc.Arguments)
			argsLine, command := tool.DisplaySummary(tc.Name, rawArgs)
			if cb != nil && cb.OnToolStart != nil {
				cb.OnToolStart(tc.Name, argsLine, command)
			}
			body := r.executeTool(ctx, sessionID, tc)
			displayResult, isError := ctxpkg.UnpackToolBody(body)
			if cb != nil && cb.OnToolEnd != nil {
				cb.OnToolEnd(tc.Name, argsLine, command, displayResult, isError)
			}
			body = ctxpkg.TruncateToolResult(body, r.Cfg)
			if err := r.Sessions.AppendMessage(ctx, session.Message{
				SessionID:  sessionID,
				Role:       role.Tool,
				Content:    body,
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
			}); err != nil {
				return nil, err
			}
		}
		if cb != nil && cb.OnContentDelta != nil && roundText != "" && !streamedContent {
			cb.OnContentDelta(roundText)
		}
		if cb != nil && cb.OnAssistantSegmentEnd != nil {
			cb.OnAssistantSegmentEnd()
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
	out, err := r.Tools.Execute(ctx, tc.Name, rawArgs)
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
