package prompt

import "strings"

// MergeSystem combines fixed system base with project context into one system string.
func MergeSystem(systemBase, agentsMD, rules, skills, gitSnapshot string) string {
	var b strings.Builder
	base := systemBase
	if strings.TrimSpace(base) == "" {
		base = DefaultSystemBase
	}
	b.WriteString(strings.TrimSpace(base))
	appendSection(&b, SectionAgentsMD, agentsMD)
	appendSection(&b, SectionRules, rules)
	appendSection(&b, SectionSkill, skills)
	appendSection(&b, SectionGit, gitSnapshot)
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
