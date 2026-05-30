package agentdef

import "github.com/wzhejunqiu/ds-code/internal/tool"

// Definition describes a built-in agent type.
type Definition struct {
	Type            AgentType
	Description     string         // when-to-use summary shown in tool schema
	Tools           []string       // tool pool spec (["*"] = all allowed)
	DisallowedTools []tool.Name    // tools forbidden for this type
	Model           ModelSelection // ModelInherit uses subagent config → main model chain
	ForceBackground bool           // always run async (verification)
	PermissionMode  AgentPermMode  // AgentPermModeInherit, AgentPermModeReadonly, AgentPermModeBubble
	ReadOnly        bool           // entire agent is read-only (Explore, Plan)
	OmitHeavyRules  bool           // skip AGENTS.md / rules injection
	PromptOverlay   string         // injected into child agent system prompt (empty = none)
}
