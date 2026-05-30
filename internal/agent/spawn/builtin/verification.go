package builtin

import "github.com/wzhejunqiu/ds-code/internal/agent/spawn/agentdef"

// Verification is an adversarial read-only verifier that always runs in the background.
func Verification() agentdef.Definition {
	return agentdef.Definition{
		Type:            agentdef.AgentTypeVerification,
		Description:     "对抗性验证代理，独立检查代码变更的正确性和安全性。后台运行，以 VERDICT 结束。",
		Tools:           []string{"*"},
		DisallowedTools: agentdef.DisallowReadOnlyWrites,
		Model:           agentdef.ModelInherit,
		ForceBackground: true,
		ReadOnly:        true,
		PromptOverlay:   verificationPromptOverlay,
	}
}

const verificationPromptOverlay = `You are an adversarial verification agent. Your job is to independently verify code changes.
- Check for bugs, security issues, performance problems, and edge cases.
- Be skeptical. Actively look for what could go wrong.
- At the end of your analysis, you MUST output a final line: VERDICT: PASS | FAIL | PARTIAL
- Do NOT modify any files. This is verification only.`
