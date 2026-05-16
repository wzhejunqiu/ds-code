package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/deepseek"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// Runner executes the agent loop.
type Runner struct {
	LLM     llm.Client
	Tools   *tool.Registry
	Perm    *permission.Engine
	Sessions session.Store
	Context *ctxpkg.Service
	Cfg     *config.Config
	MaxTurns int
	Out     io.Writer
}

// TurnResult is the outcome of a user turn.
type TurnResult struct {
	FinalContent   string
	FinalReasoning string
	Usage          llm.Usage
	SubRounds      int
}

// RunTurn handles one user message through sub-rounds until no tool_calls or max turns.
func (r *Runner) RunTurn(ctx context.Context, sessionID, userText string) (*TurnResult, error) {
	expanded, err := r.Context.ExpandUserText(userText)
	if err != nil {
		return nil, fmt.Errorf("expand @ references: %w", err)
	}
	if err := r.Sessions.AppendMessage(ctx, session.Message{
		SessionID: sessionID,
		Role:      "user",
		Content:   expanded,
	}); err != nil {
		return nil, err
	}

	sess, err := r.Sessions.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	result := &TurnResult{}
	for round := 0; round < r.MaxTurns; round++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

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

		resp, err := r.LLM.Chat(ctx, req)
		if err != nil {
			if deepseek.IsContextTooLong(err) {
				_ = r.Context.CompactAPIContext(ctx, sessionID)
				continue
			}
			return nil, err
		}

		_ = r.Sessions.AddUsage(ctx, sessionID, resp.Usage)
		result.Usage = resp.Usage
		result.SubRounds = round + 1

		tcJSON, _ := json.Marshal(resp.ToolCalls)
		if err := r.Sessions.AppendMessage(ctx, session.Message{
			SessionID:        sessionID,
			Role:             "assistant",
			Content:          resp.Content,
			ReasoningContent: resp.ReasoningContent,
			ToolCallsJSON:    string(tcJSON),
		}); err != nil {
			return nil, err
		}

		if len(resp.ToolCalls) == 0 {
			result.FinalContent = resp.Content
			result.FinalReasoning = resp.ReasoningContent
			if r.Out != nil && resp.Content != "" {
				_, _ = io.WriteString(r.Out, resp.Content)
			}
			return result, nil
		}

		for _, tc := range resp.ToolCalls {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			body := r.executeTool(ctx, tc)
			body = ctxpkg.TruncateToolResult(body, r.Cfg)
			if err := r.Sessions.AppendMessage(ctx, session.Message{
				SessionID:  sessionID,
				Role:       "tool",
				Content:    body,
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
			}); err != nil {
				return nil, err
			}
		}
	}
	return nil, fmt.Errorf("agent: exceeded max sub-rounds (%d)", r.MaxTurns)
}

func (r *Runner) executeTool(ctx context.Context, tc llm.ToolCall) string {
	args := tool.ArgsMap([]byte(tc.Arguments))
	argMap := args
	if err := r.Perm.Check(tc.Name, argMap); err != nil {
		return ctxpkg.FormatToolError(tc.Name, tc.ID, err)
	}
	out, err := r.Tools.Execute(ctx, tc.Name, []byte(tc.Arguments))
	if err != nil {
		return ctxpkg.FormatToolError(tc.Name, tc.ID, err)
	}
	return ctxpkg.FormatToolResult(tc.Name, tc.ID, out)
}

func cacheScope(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return hex.EncodeToString(sum[:])
}
