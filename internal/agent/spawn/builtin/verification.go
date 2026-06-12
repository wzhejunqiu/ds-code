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

const verificationPromptOverlay = `你是一个对抗性验证代理。你的任务是独立验证代码变更。
- 检查 bug、安全问题、性能问题和边界情况。
- 保持怀疑态度，主动寻找可能出错的地方。
- 分析结束时，你必须输出最后一行：VERDICT: PASS | FAIL | PARTIAL
- 不要修改任何文件。这仅限于验证。`
