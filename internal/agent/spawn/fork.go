package spawn

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

// ForkPlaceholder is the byte-identical text inserted for every tool_result in a fork child.
// It must NOT vary between children so that the message prefix stays cache-identical.
const ForkPlaceholder = "Fork started — processing in background"

const forkBoilerplateTag = "fork-boilerplate"

// ForkBoilerplate is the behavioral rules injected into every fork child message.
const ForkBoilerplate = `<fork-boilerplate>
You are a forked subagent. Follow these rules:

1. STOP and read the full context. Understand the task before acting.
2. Do NOT spawn sub-agents. Use tools directly.
3. Commit all changes you make unless told otherwise.
4. Do NOT ask questions, make conversation, or suggest next steps. Just do the work.
5. When you're done, provide a concise report.

Output format:
Scope: [what was asked]
Result: [what you did]
Key files: [files you read or modified]
Files changed: [list with brief description per file]
Issues: [anything unresolved, or "None"]
</fork-boilerplate>`

// BuildForkMessages constructs the child agent conversation from the parent's
// API messages. It clones the triggering assistant message (all tool_use blocks)
// and appends a single user message with identical placeholder tool_results
// plus the fork directive.
func BuildForkMessages(parentMessages []llm.Message, parentToolCalls []llm.ToolCall, directive string) []llm.Message {
	out := make([]llm.Message, 0, len(parentMessages)+2)

	// Copy history up to (but not including) the triggering assistant message.
	// Find the last assistant message that contains the agent tool_call.
	triggerIdx := -1
	for i := len(parentMessages) - 1; i >= 0; i-- {
		if parentMessages[i].Role == role.Assistant && len(parentMessages[i].ToolCalls) > 0 {
			triggerIdx = i
			break
		}
	}

	if triggerIdx >= 0 {
		out = append(out, parentMessages[:triggerIdx+1]...)
	} else if len(parentMessages) > 0 {
		// Skill fork and other non-agent triggers: include full parent API context.
		out = append(out, parentMessages...)
	}

	// Build child user message: placeholder tool_results (agent fork only) + boilerplate + directive
	var sb strings.Builder
	if len(parentToolCalls) > 0 {
		for _, tc := range parentToolCalls {
			sb.WriteString(fmt.Sprintf("\n<tool_result tool_call_id=%q>\n%s\n</tool_result>\n", tc.ID, ForkPlaceholder))
		}
	}
	sb.WriteString("\n" + ForkBoilerplate + "\n\n")
	sb.WriteString(buildChildDirective(directive))

	out = append(out, llm.Message{
		Role:    role.User,
		Content: sb.String(),
	})
	return out
}

// IsInForkChild checks whether the messages contain a fork boilerplate tag,
// indicating this is already a fork child (prevent recursive fork).
func IsInForkChild(messages []llm.Message) bool {
	for _, m := range messages {
		if m.Role == role.User && strings.Contains(m.Content, forkBoilerplateTag) {
			return true
		}
	}
	return false
}

// buildChildDirective wraps the task prompt in the plan-mandated directive tag.
func buildChildDirective(prompt string) string {
	encoded, err := json.Marshal(prompt)
	if err != nil {
		encoded = []byte(`"` + strings.ReplaceAll(prompt, `"`, `\"`) + `"`)
	}
	return fmt.Sprintf(`[directive: "Here's your specific task: %s"]`, string(encoded))
}
