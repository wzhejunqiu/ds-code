package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
	"go.uber.org/zap"
)

type subRoundStream struct {
	timing          streamTiming
	roundContent    strings.Builder
	streamedContent bool
	planningDone    bool
}

func (r *Runner) chatWithCompactRetry(ctx context.Context, sessionID string, req llm.Request) (*llm.Response, error) {
	resp, err := r.LLM.Chat(ctx, req)
	if err == nil || !llm.IsContextTooLong(err) {
		return resp, err
	}
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
	return r.LLM.Chat(ctx, req)
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
	return result, nil
}

func (r *Runner) runToolCalls(
	ctx context.Context,
	sessionID string,
	toolCalls []llm.ToolCall,
	resp *llm.Response,
	stream *subRoundStream,
	cb *TurnCallbacks,
) error {
	for _, tc := range toolCalls {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logging.L().Info("tool execute",
			zap.String("session_id", sessionID),
			zap.String("tool", tc.Name),
			zap.String("call_id", tc.ID),
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
		body := r.executeTool(ctx, sessionID, tc)
		displayResult, isError := ctxpkg.UnpackToolBody(body)
		endRows := tool.ToolEndRows(tc.Name, rawArgs, r.Perm.Workspace)
		if len(endRows) == 0 {
			argsLine, command := tool.DisplaySummary(tc.Name, rawArgs, r.Perm.Workspace)
			if tc.Name == "read_file" && !isError {
				if start, end, ok := tool.ReadFileLineRange(displayResult); ok {
					argsLine = tool.AppendReadFileLineRange(argsLine, start, end)
				}
			}
			if tc.Name == "grep" && !isError {
				argsLine = tool.AppendGrepResultSuffix(argsLine, displayResult)
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
		body = ctxpkg.TruncateToolResult(body, r.Cfg)
		if err := r.Sessions.AppendMessage(ctx, session.Message{
			SessionID:  sessionID,
			Role:       role.Tool,
			Content:    body,
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
		}); err != nil {
			return err
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
	resp *llm.Response,
	stream *subRoundStream,
) error {
	tcJSON, err := json.Marshal(resp.ToolCalls)
	if err != nil {
		return fmt.Errorf("marshal tool_calls: %w", err)
	}
	assistantMsg := session.Message{
		SessionID:           sessionID,
		Role:                role.Assistant,
		Content:             resp.Content,
		ReasoningContent:    resp.ReasoningContent,
		ReasoningDurationMS: durationMS(stream.timing.duration()),
		ToolCallsJSON:       string(tcJSON),
	}
	return r.Sessions.AppendMessage(ctx, assistantMsg)
}
