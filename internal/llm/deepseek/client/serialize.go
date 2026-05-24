package client

import (
	"encoding/json"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/prompt"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

// MergeSystem combines fixed system base with project context into one system string.
func MergeSystem(systemBase, runtimeEnv, agentsMD, rules, skills, gitSnapshot string) string {
	return prompt.MergeSystem(systemBase, runtimeEnv, agentsMD, rules, skills, gitSnapshot, "")
}

// ToAPIMessages converts view fields to API messages (single system + history).
func ToAPIMessages(mergedSystem string, messages []llm.Message) []map[string]any {
	out := make([]map[string]any, 0, 1+len(messages))
	if mergedSystem != "" {
		out = append(out, map[string]any{
			"role":    "system",
			"content": mergedSystem,
		})
	}
	for _, m := range messages {
		msg := map[string]any{"role": m.Role}
		switch m.Role {
		case role.Assistant:
			if m.Content != "" {
				msg["content"] = m.Content
			}
			if m.ReasoningContent != "" {
				msg["reasoning_content"] = m.ReasoningContent
			}
			if len(m.ToolCalls) > 0 {
				msg["tool_calls"] = serializeToolCalls(m.ToolCalls)
			}
		case role.Tool:
			msg["content"] = m.Content
			msg["tool_call_id"] = m.ToolCallID
			if m.Name != "" {
				msg["name"] = m.Name
			}
		default:
			msg["content"] = m.Content
		}
		out = append(out, msg)
	}
	return out
}

func serializeToolCalls(calls []llm.ToolCall) []map[string]any {
	out := make([]map[string]any, len(calls))
	for i, tc := range calls {
		out[i] = map[string]any{
			"id":   tc.ID,
			"type": "function",
			"function": map[string]any{
				"name":      tc.Name,
				"arguments": tc.Arguments,
			},
		}
	}
	return out
}

// ToolsToAPI converts tool definitions for the API.
func ToolsToAPI(tools []llm.ToolDef, strict bool) []map[string]any {
	out := make([]map[string]any, len(tools))
	for i, t := range tools {
		fn := map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.Parameters,
		}
		if strict {
			fn["strict"] = true
		}
		out[i] = map[string]any{
			"type":     "function",
			"function": fn,
		}
	}
	return out
}

// ToolsJSON serializes tool defs for CountBreakdown (Phase 3+).
func ToolsJSON(tools []llm.ToolDef) string {
	b, _ := json.Marshal(ToolsToAPI(tools, false))
	return string(b)
}
