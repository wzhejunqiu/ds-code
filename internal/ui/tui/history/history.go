// Package history maps persisted session messages to TUI chat blocks.
package history

import (
	"context"
	"encoding/json"
	"time"

	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
)

// BlocksFromMessages builds TUI chat blocks from persisted session messages.
func BlocksFromMessages(msgs []session.Message, reasoningOpen bool, workspace string) []chat.Block {
	var blocks []chat.Block
	for i := 0; i < len(msgs); i++ {
		msg := msgs[i]
		switch msg.Role {
		case role.User:
			b := chat.Block{Role: chat.RoleUser}
			b.Content.WriteString(msg.Content)
			blocks = append(blocks, b)
		case role.Assistant:
			calls := parseToolCalls(msg.ToolCallsJSON)
			if msg.Content != "" || msg.ReasoningContent != "" {
				b := chat.Block{Role: chat.RoleAssistant, ReasoningOpen: reasoningOpen}
				b.Content.WriteString(msg.Content)
				b.Reasoning.WriteString(msg.ReasoningContent)
				if msg.ReasoningDurationMS > 0 {
					b.ReasoningDuration = time.Duration(msg.ReasoningDurationMS) * time.Millisecond
				}
				if msg.TurnDurationMS > 0 {
					b.TurnDuration = time.Duration(msg.TurnDurationMS) * time.Millisecond
				}
				blocks = append(blocks, b)
			}
			if len(calls) > 0 {
				for _, tc := range calls {
					toolMsg := findToolMessage(msgs, i+1, tc.ID)
					result, isError := "", false
					if toolMsg != nil {
						result, isError = ctxpkg.UnpackToolBody(toolMsg.Content)
					}
					argsLine, command := tool.DisplaySummary(tc.Name, []byte(tc.Arguments), workspace)
					blocks = append(blocks, chat.Block{
						Role:        chat.RoleTool,
						ToolName:    tc.Name,
						ToolCallID:  tc.ID,
						ToolArgs:    argsLine,
						ToolCommand: command,
						ToolResult:  result,
						ToolError:   isError,
					})
				}
			}
		case role.Tool:
		case role.System:
			if msg.Content == chat.InterruptSessionMarker() {
				blocks = append(blocks, chat.Block{Role: chat.RoleInterrupt})
			}
		}
	}
	return blocks
}

func parseToolCalls(raw string) []llm.ToolCall {
	raw = trimJSON(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return nil
	}
	var calls []llm.ToolCall
	if err := json.Unmarshal([]byte(raw), &calls); err != nil {
		return nil
	}
	return calls
}

func trimJSON(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\t') {
		s = s[1:]
	}
	return s
}

func findToolMessage(msgs []session.Message, start int, callID string) *session.Message {
	for i := start; i < len(msgs); i++ {
		if msgs[i].Role != role.Tool {
			break
		}
		if msgs[i].ToolCallID == callID {
			return &msgs[i]
		}
	}
	return nil
}

// LoadSessionChat loads all messages for a session and converts them for the TUI.
func LoadSessionChat(store session.Store, sessionID string, reasoningOpen bool, workspace string) ([]chat.Block, error) {
	msgs, err := store.ListMessages(context.Background(), sessionID)
	if err != nil {
		return nil, err
	}
	return BlocksFromMessages(msgs, reasoningOpen, workspace), nil
}
