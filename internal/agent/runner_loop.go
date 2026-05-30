package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"go.uber.org/zap"
)

type subRoundStream struct {
	timing          streamTiming
	roundContent    strings.Builder
	streamedContent bool
	planningDone    bool
}

func (r *Runner) attachStreamHandlers(cb *TurnCallbacks, round int, stream *subRoundStream) func(llm.StreamDelta) {
	if cb == nil {
		return nil
	}
	endPlanning := func() {
		if stream.planningDone || cb.OnPlanningEnd == nil {
			return
		}
		stream.planningDone = true
		cb.OnPlanningEnd()
	}
	return func(d llm.StreamDelta) {
		stream.timing.observe(d)
		if d.Content != "" || d.Reasoning != "" {
			endPlanning()
		}
		if d.Content != "" {
			stream.roundContent.WriteString(d.Content)
			if cb.OnContentDelta != nil {
				stream.streamedContent = true
				cb.OnContentDelta(d.Content)
			}
		}
		if d.Reasoning != "" && cb.OnReasoningDelta != nil {
			cb.OnReasoningDelta(d.Reasoning)
		}
	}
}

func (r *Runner) finishTerminalRound(
	ctx context.Context,
	sessionID string,
	modelID string,
	resp *llm.Response,
	stream *subRoundStream,
	turnStart time.Time,
	result *TurnResult,
	cb *TurnCallbacks,
) (*TurnResult, error) {
	turnDur := time.Since(turnStart)
	reasoningDur := stream.timing.duration()
	assistantMsg := session.Message{
		SessionID:           sessionID,
		Role:                role.Assistant,
		Content:             resp.Content,
		ReasoningContent:    resp.ReasoningContent,
		ReasoningDurationMS: durationMS(reasoningDur),
		TurnDurationMS:      durationMS(turnDur),
	}
	enrichAssistantUsage(&assistantMsg, modelID, resp.Usage)
	if err := r.Sessions.AppendMessage(ctx, assistantMsg); err != nil {
		return nil, err
	}

	result.FinalContent = resp.Content
	result.FinalReasoning = resp.ReasoningContent
	result.TurnDuration = turnDur
	result.FinalReasoningDuration = reasoningDur

	if cb != nil && cb.OnContentDelta != nil && !stream.streamedContent {
		content := resp.Content
		if stream.roundContent.Len() > 0 {
			content = stream.roundContent.String()
		}
		if content != "" {
			cb.OnContentDelta(content)
		}
	}
	logging.L().Info("user turn done (no tools)",
		zap.String("session_id", sessionID),
	)
	if cb == nil && r.Out != nil && resp.Content != "" {
		_, _ = io.WriteString(r.Out, resp.Content)
	}
	if r.Hooks != nil {
		r.Hooks.Run(ctx, HookStop, marshalHookInput(HookInput{SessionID: sessionID}))
	}
	return result, nil
}

func (r *Runner) finishMaxTurnsExceeded(
	ctx context.Context,
	sessionID string,
	sess session.Session,
	turnStart time.Time,
	result *TurnResult,
	state *LoopState,
	cb *TurnCallbacks,
) (*TurnResult, error) {
	logging.L().Warn("exceeded max sub-rounds", zap.String("session_id", sessionID), zap.Int("max", r.MaxTurns))
	state.Transition = TransMaxTurns

	if err := r.Sessions.AppendMessage(ctx, session.Message{
		SessionID: sessionID,
		Role:      role.System,
		Content:   fmt.Sprintf(maxTurnsSystemEventFmt, r.MaxTurns),
	}); err != nil {
		return nil, err
	}
	if err := r.Sessions.AppendMessage(ctx, session.Message{
		SessionID: sessionID,
		Role:      role.User,
		Content:   maxTurnsSummaryPrompt,
	}); err != nil {
		return nil, err
	}

	view, maxTokens, err := r.Context.PrepareRequest(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	stream := &subRoundStream{}
	req := llm.Request{
		MergedSystem:    view.MergedSystem(),
		Messages:        view.Messages,
		Model:           sess.Model,
		MaxTokens:       maxTokens,
		Stream:          true,
		ThinkingType:    sess.ThinkingType,
		ReasoningEffort: sess.ReasoningEffort,
		UserID:          cacheScope(sessionID),
		StrictTools:     r.Cfg.LLM.StrictTools,
	}
	req.OnStream = r.attachStreamHandlers(cb, r.MaxTurns, stream)

	resp, err := r.chatWithRecovery(ctx, sessionID, req, state)
	if err != nil {
		return nil, err
	}
	if err := r.Sessions.AddUsage(ctx, sessionID, resp.Usage); err != nil {
		logging.L().Warn("add usage failed", zap.String("session_id", sessionID), zap.Error(err))
	}
	result.Usage = resp.Usage
	return r.finishTerminalRound(ctx, sessionID, sess.Model, resp, stream, turnStart, result, cb)
}

func (r *Runner) runToolCalls(
	ctx context.Context,
	sessionID string,
	toolCalls []llm.ToolCall,
	resp *llm.Response,
	stream *subRoundStream,
	cb *TurnCallbacks,
) error {
	batches := partitionToolCalls(r.Tools, toolCalls)

	for _, batch := range batches {
		for _, tc := range batch.calls {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logging.L().Info("tool execute",
				zap.String("session_id", sessionID),
				zap.String("tool", tc.Name),
				zap.String("call_id", tc.ID),
				zap.Bool("concurrent", batch.concurrent),
			)
			rawArgs := []byte(tc.Arguments)
			patchDisplays := tool.ApplyPatchStarts(tc.Name, rawArgs, r.Perm.Workspace)
			if cb != nil && cb.OnToolStart != nil {
				if len(patchDisplays) > 0 {
					for _, d := range patchDisplays {
						cb.OnToolStart(tc.Name, d.Args, d.Command)
					}
				} else {
					argsLine, command := tool.DisplaySummary(tc.Name, rawArgs, r.Perm.Workspace)
					cb.OnToolStart(tc.Name, argsLine, command)
				}
			}
		}

		if batch.concurrent {
			if err := r.runConcurrentBatch(ctx, sessionID, batch.calls); err != nil {
				return err
			}
		} else {
			if err := r.runSerialBatch(ctx, sessionID, batch.calls); err != nil {
				return err
			}
		}

		for _, tc := range batch.calls {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			rawArgs := []byte(tc.Arguments)
			msgs, _ := r.Sessions.ListMessages(ctx, sessionID)
			displayResult, isError := findToolResult(msgs, tc.ID)
			endRows := tool.ToolEndRows(tc.Name, rawArgs, r.Perm.Workspace)
			if len(endRows) == 0 {
				argsLine, command := tool.DisplaySummary(tc.Name, rawArgs, r.Perm.Workspace)
				if tc.Name == "read_file" && !isError {
					if start, end, ok := tool.ReadFileLineRange(displayResult); ok {
						argsLine = tool.AppendReadFileLineRange(argsLine, start, end)
					}
				}
				if tc.Name == "grep" && !isError {
					argsLine = tool.AppendGrepResultSuffix(argsLine, rawArgs, displayResult)
				}
				if (tc.Name == "glob" || tc.Name == "list_dir") && !isError {
					argsLine = tool.AppendPathResultSuffix(argsLine, displayResult)
				}
				if cb != nil && cb.OnToolEnd != nil {
					cb.OnToolEnd(tc.Name, argsLine, command, displayResult, isError)
				}
			} else {
				for _, row := range endRows {
					argsLine := row.Args
					if tc.Name == "read_file" && !isError {
						if start, end, ok := tool.ReadFileLineRange(displayResult); ok {
							argsLine = tool.AppendReadFileLineRange(argsLine, start, end)
						}
					}
					if cb != nil && cb.OnToolEnd != nil {
						cb.OnToolEnd(tc.Name, argsLine, row.Command, displayResult, isError)
					}
				}
			}
		}
	}

	roundText := resp.Content
	if stream.roundContent.Len() > 0 {
		roundText = stream.roundContent.String()
	}
	if cb != nil && cb.OnContentDelta != nil && roundText != "" && !stream.streamedContent {
		cb.OnContentDelta(roundText)
	}
	if cb != nil && cb.OnAssistantSegmentEnd != nil {
		cb.OnAssistantSegmentEnd()
	}
	return nil
}

func (r *Runner) appendAssistantWithTools(
	ctx context.Context,
	sessionID string,
	modelID string,
	resp *llm.Response,
	stream *subRoundStream,
) error {
	tcJSON, err := json.Marshal(resp.ToolCalls)
	if err != nil {
		return fmt.Errorf("marshal tool_calls: %w", err)
	}
	logging.L().Debug("append assistant with tools",
		zap.String("session_id", sessionID),
		zap.Int("tool_calls", len(resp.ToolCalls)),
		zap.Int("content_chars", len(resp.Content)),
	)
	assistantMsg := session.Message{
		SessionID:           sessionID,
		Role:                role.Assistant,
		Content:             resp.Content,
		ReasoningContent:    resp.ReasoningContent,
		ReasoningDurationMS: durationMS(stream.timing.duration()),
		ToolCallsJSON:       string(tcJSON),
	}
	enrichAssistantUsage(&assistantMsg, modelID, resp.Usage)
	return r.Sessions.AppendMessage(ctx, assistantMsg)
}

func findToolResult(msgs []session.Message, toolCallID string) (string, bool) {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].ToolCallID == toolCallID {
			return ctxpkg.UnpackToolBody(msgs[i].Content)
		}
	}
	return "", false
}
