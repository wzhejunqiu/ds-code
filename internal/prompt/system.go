package prompt

import "strings"

// MergeSystemStatic is the cache-stable prefix: base prompt + sorted tool JSON.
func MergeSystemStatic(systemBase, toolsJSON string) string {
	var b strings.Builder
	base := systemBase
	if strings.TrimSpace(base) == "" {
		base = DefaultSystemBase
	}
	b.WriteString(strings.TrimSpace(base))
	appendSection(&b, SectionTools, toolsJSON)
	return b.String()
}

// MergeSystemDynamic is the per-turn variable suffix of the system prompt.
func MergeSystemDynamic(runtimeEnv, agentsMD, rules, skills, gitSnapshot, agentOverlay string) string {
	var b strings.Builder
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

// MergeSystem combines fixed system base with project context into one system string.
func MergeSystem(systemBase, runtimeEnv, agentsMD, rules, skills, gitSnapshot, agentOverlay string) string {
	return MergeSystemParts(
		MergeSystemStatic(systemBase, ""),
		MergeSystemDynamic(runtimeEnv, agentsMD, rules, skills, gitSnapshot, agentOverlay),
	)
}

// MergeSystemParts joins static and dynamic system sections.
func MergeSystemParts(static, dynamic string) string {
	return static + dynamic
}

func appendSection(b *strings.Builder, header, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	b.WriteString(header)
	b.WriteString(body)
}
