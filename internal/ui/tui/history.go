// Session history → in-memory chat transcript (history.go).
// Persisted rows live in session.Store; this package only maps Message → chatBlock.
package tui

import (
	"context"
	"encoding/json"
	"time"

	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// chatBlocksFromMessages builds TUI chat blocks from persisted session messages.
//
// Mapping rules:
//   - user → chatRoleUser
//   - assistant (content/reasoning) → chatRoleAssistant; tool_calls expand to chatRoleTool rows
//   - tool → skipped here (paired via assistant ToolCallsJSON + findToolMessage)
//   - system → chatRoleInterrupt only when content matches interruptSessionMarker()
//
// Orphan tool messages (no matching call id) are ignored. Generic system rows are omitted.
func chatBlocksFromMessages(msgs []session.Message, reasoningOpen bool) []chatBlock {
	var blocks []chatBlock
	for i := 0; i < len(msgs); i++ {
		msg := msgs[i]
		switch msg.Role {
		case role.User:
			b := chatBlock{role: chatRoleUser}
			b.content.WriteString(msg.Content)
			blocks = append(blocks, b)
		case role.Assistant:
			calls := parseToolCalls(msg.ToolCallsJSON)
			if msg.Content != "" || msg.ReasoningContent != "" {
				b := chatBlock{role: chatRoleAssistant, reasoningOpen: reasoningOpen}
				b.content.WriteString(msg.Content)
				b.reasoning.WriteString(msg.ReasoningContent)
				if msg.ReasoningDurationMS > 0 {
					b.reasoningDuration = time.Duration(msg.ReasoningDurationMS) * time.Millisecond
				}
				if msg.TurnDurationMS > 0 {
					b.turnDuration = time.Duration(msg.TurnDurationMS) * time.Millisecond
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
					argsLine, command := tool.DisplaySummary(tc.Name, []byte(tc.Arguments))
					blocks = append(blocks, chatBlock{
						role:        chatRoleTool,
						toolName:    tc.Name,
						toolArgs:    argsLine,
						toolCommand: command,
						toolResult:  result,
						toolError:   isError,
					})
				}
			}
		case role.Tool:
			// rendered with the preceding assistant tool_calls row
		case role.System:
			if msg.Content == interruptSessionMarker() {
				blocks = append(blocks, chatBlock{role: chatRoleInterrupt})
			}
		}
	}
	return blocks
}

// parseToolCalls decodes assistant ToolCallsJSON; empty/invalid → nil.
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

// findToolMessage scans consecutive tool rows after an assistant message for a call id.
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

// loadSessionChat loads all messages for a session and converts them for the TUI.
// Used on startup (loadInitialHistory) and /resume (resumeSession).
func loadSessionChat(store session.Store, sessionID string, reasoningOpen bool) ([]chatBlock, error) {
	msgs, err := store.ListMessages(context.Background(), sessionID)
	if err != nil {
		return nil, err
	}
	return chatBlocksFromMessages(msgs, reasoningOpen), nil
}
