package context

import (
	"encoding/json"

	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/session"
)

func messageToLLM(m session.Message) llm.Message {
	switch m.Role {
	case "user":
		return llm.Message{Role: "user", Content: m.Content}
	case "assistant":
		am := llm.Message{
			Role:             "assistant",
			Content:          m.Content,
			ReasoningContent: m.ReasoningContent,
		}
		if m.ToolCallsJSON != "" {
			var calls []llm.ToolCall
			_ = json.Unmarshal([]byte(m.ToolCallsJSON), &calls)
			am.ToolCalls = calls
		}
		return am
	case "tool":
		return llm.Message{
			Role:       "tool",
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			Name:       m.ToolName,
		}
	default:
		return llm.Message{Role: m.Role, Content: m.Content}
	}
}

func compactSummaryMessage(summary string) llm.Message {
	return llm.Message{
		Role:    "assistant",
		Content: "[Conversation summary]\n" + summary,
	}
}
