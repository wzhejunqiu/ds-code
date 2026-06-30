package agent

import (
	"context"
	"sync"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/shell"
)

const maxConcurrentReadTools = 10

// toolBatch holds a group of tool calls to execute together.
type toolBatch struct {
	concurrent bool
	calls      []llm.ToolCall
}

// partitionToolCalls splits tool_calls into batches: adjacent read-only tools
// form concurrent batches; write tools each get their own serial batch.
func partitionToolCalls(reg *tool.Registry, calls []llm.ToolCall) []toolBatch {
	if len(calls) == 0 {
		return nil
	}
	var batches []toolBatch
	var pending []llm.ToolCall

	flush := func() {
		if len(pending) == 0 {
			return
		}
		batches = append(batches, toolBatch{concurrent: true, calls: pending})
		pending = nil
	}

	for _, tc := range calls {
		if isConcurrentToolCall(reg, tc) {
			pending = append(pending, tc)
			if len(pending) >= maxConcurrentReadTools {
				flush()
			}
		} else {
			flush()
			batches = append(batches, toolBatch{concurrent: false, calls: []llm.ToolCall{tc}})
		}
	}
	flush()
	return batches
}

func isConcurrentToolCall(reg *tool.Registry, tc llm.ToolCall) bool {
	t, ok := reg.Get(tc.Name)
	if !ok {
		return false
	}
	if tool.IsToolConcurrencySafe(t) && tool.IsToolReadOnly(t) {
		return true
	}
	return tool.NameShell.Matches(tc.Name) && shell.IsBackgroundArgs([]byte(tc.Arguments))
}

func (r *Runner) runConcurrentBatch(ctx context.Context, sessionID string, calls []llm.ToolCall) error {
	if len(calls) == 0 {
		return nil
	}
	if len(calls) == 1 {
		body := r.executeSingleTool(ctx, sessionID, calls[0])
		return r.persistToolResult(ctx, sessionID, calls[0], body)
	}

	results := make([]string, len(calls))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentReadTools)

	for i, tc := range calls {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		wg.Add(1)
		go func(idx int, call llm.ToolCall) {
			defer wg.Done()
			defer logging.Bind(ctx)()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = r.executeSingleTool(ctx, sessionID, call)
		}(i, tc)
	}
	wg.Wait()

	for i, tc := range calls {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := r.persistToolResult(ctx, sessionID, tc, results[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) runSerialBatch(ctx context.Context, sessionID string, calls []llm.ToolCall) error {
	for _, tc := range calls {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		body := r.executeSingleTool(ctx, sessionID, tc)
		if err := r.persistToolResult(ctx, sessionID, tc, body); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) executeSingleTool(ctx context.Context, sessionID string, tc llm.ToolCall) string {
	body := r.executeTool(ctx, sessionID, tc)
	return r.finalizeToolResult(sessionID, tc, body)
}

func (r *Runner) persistToolResult(ctx context.Context, sessionID string, tc llm.ToolCall, body string) error {
	return r.Sessions.AppendMessage(ctx, session.Message{
		SessionID:  sessionID,
		Role:       "tool",
		Content:    body,
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
	})
}
