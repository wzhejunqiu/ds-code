package prompt

import "strings"

// MergeSystem combines fixed system base with project context into one system string.
func MergeSystem(systemBase, runtimeEnv, agentsMD, rules, skills, gitSnapshot, agentOverlay string) string {
	var b strings.Builder
	base := systemBase
	if strings.TrimSpace(base) == "" {
		base = DefaultSystemBase
	}
	b.WriteString(strings.TrimSpace(base))
	appendSection(&b, SectionRuntimeEnv, runtimeEnv)
	appendSection(&b, SectionAgentsMD, agentsMD)
	appendSection(&b, SectionRules, rules)
	appendSection(&b, SectionAgentOverlay, agentOverlay)
	appendSection(&b, SectionSkill, skills)
	appendSection(&b, SectionGit, gitSnapshot)
	if agentOverlay != "" {
		b.WriteString("\n</agent-type-overlay>")
	}
	return b.String()
}

func appendSection(b *strings.Builder, header, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	b.WriteString(header)
	b.WriteString(body)
}
