//go:build tuitest

package mockserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/tuitest/scenarios"
)

func writeStreamTurn(w http.ResponseWriter, turn *scenarios.Turn) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("mockserver: ResponseWriter is not Flusher")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)

	usage := map[string]int{"prompt_tokens": 10, "completion_tokens": 20}

	for _, ch := range turn.Chunks {
		if ch.Delay > 0 {
			time.Sleep(ch.Delay)
		}
		delta := map[string]any{}
		if ch.Content != "" {
			delta["content"] = ch.Content
		}
		if ch.Reasoning != "" {
			delta["reasoning_content"] = ch.Reasoning
		}
		if len(delta) > 0 {
			if err := writeSSE(w, map[string]any{
				"choices": []map[string]any{{"delta": delta, "finish_reason": ""}},
			}); err != nil {
				return err
			}
			flusher.Flush()
		}
	}

	if len(turn.ToolCalls) > 0 {
		tcs := make([]map[string]any, 0, len(turn.ToolCalls))
		for i, tc := range turn.ToolCalls {
			tcs = append(tcs, map[string]any{
				"index": i,
				"id":    tc.ID,
				"type":  "function",
				"function": map[string]string{
					"name":      tc.Name,
					"arguments": tc.Arguments,
				},
			})
		}
		if err := writeSSE(w, map[string]any{
			"choices": []map[string]any{{
				"delta": map[string]any{"tool_calls": tcs},
			}},
		}); err != nil {
			return err
		}
		flusher.Flush()
	}

	finish := turn.FinishReason
	if finish == "" {
		if len(turn.ToolCalls) > 0 {
			finish = "tool_calls"
		} else {
			finish = "stop"
		}
	}
	if err := writeSSE(w, map[string]any{
		"choices": []map[string]any{{"delta": map[string]any{}, "finish_reason": finish}},
		"usage":   usage,
	}); err != nil {
		return err
	}
	flusher.Flush()
	_, _ = io.WriteString(w, "data: [DONE]\n\n")
	flusher.Flush()
	return nil
}

func writeNonStreamTurn(w http.ResponseWriter, turn *scenarios.Turn) error {
	content, reasoning := aggregateText(turn.Chunks)
	calls := turn.ToolCalls
	finish := turn.FinishReason
	if finish == "" {
		if len(calls) > 0 {
			finish = "tool_calls"
		} else {
			finish = "stop"
		}
	}
	msg := map[string]any{
		"role":    "assistant",
		"content": content,
	}
	if reasoning != "" {
		msg["reasoning_content"] = reasoning
	}
	if len(calls) > 0 {
		tcs := make([]map[string]any, 0, len(calls))
		for _, tc := range calls {
			tcs = append(tcs, map[string]any{
				"id":   tc.ID,
				"type": "function",
				"function": map[string]string{
					"name":      tc.Name,
					"arguments": tc.Arguments,
				},
			})
		}
		msg["tool_calls"] = tcs
	}
	body := map[string]any{
		"choices": []map[string]any{{
			"message":       msg,
			"finish_reason": finish,
		}},
		"usage": map[string]int{"prompt_tokens": 10, "completion_tokens": 20},
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(body)
}

func aggregateText(chunks []scenarios.StreamChunk) (content, reasoning string) {
	for _, c := range chunks {
		content += c.Content
		reasoning += c.Reasoning
	}
	return content, reasoning
}

func writeSSE(w io.Writer, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", raw)
	return err
}

// DefaultStopTurn is a minimal assistant stop turn.
func DefaultStopTurn(text string) *scenarios.Turn {
	return &scenarios.Turn{
		Chunks:       []scenarios.StreamChunk{{Content: text}},
		FinishReason: "stop",
	}
}

// ToolCallTurn builds a turn that requests tools.
func ToolCallTurn(calls []llm.ToolCall) *scenarios.Turn {
	return &scenarios.Turn{
		ToolCalls:    calls,
		FinishReason: "tool_calls",
	}
}
