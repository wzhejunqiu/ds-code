package context

import (
	"encoding/json"

	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
)

func messageToLLM(m session.Message) llm.Message {
	switch m.Role {
	case role.User:
		return llm.Message{Role: role.User, Content: m.Content}
	case role.Assistant:
		am := llm.Message{
			Role:             role.Assistant,
			Content:          m.Content,
			ReasoningContent: m.ReasoningContent,
		}
		if m.ToolCallsJSON != "" {
			var calls []llm.ToolCall
			_ = json.Unmarshal([]byte(m.ToolCallsJSON), &calls)
			am.ToolCalls = calls
		}
		return am
	case role.Tool:
		return llm.Message{
			Role:       role.Tool,
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
		Role:    role.Assistant,
		Content: ConversationSummaryPrefix + summary,
	}
}
