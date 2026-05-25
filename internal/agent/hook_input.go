package agent

import (
	"encoding/json"

	"github.com/wzhejunqiu/ds-code/internal/llm"
)

// HookInput is JSON-serialized into HOOK_INPUT for hook scripts.
type HookInput struct {
	SessionID  string `json:"session_id,omitempty"`
	Tool       string `json:"tool,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	Args       string `json:"args,omitempty"`
	Error      string `json:"error,omitempty"`
	AgentID    string `json:"agent_id,omitempty"`
	AgentType  string `json:"agent_type,omitempty"`
}

// MarshalHookInput serializes hook payload for HOOK_INPUT.
func MarshalHookInput(in HookInput) string {
	b, _ := json.Marshal(in)
	return string(b)
}

func marshalHookInput(in HookInput) string {
	return MarshalHookInput(in)
}

func hookInputForTool(sessionID string, tc llm.ToolCall, args []byte, execErr error) string {
	in := HookInput{
		SessionID:  sessionID,
		Tool:       tc.Name,
		ToolCallID: tc.ID,
		Args:       string(args),
	}
	if execErr != nil {
		in.Error = execErr.Error()
	}
	return marshalHookInput(in)
}
