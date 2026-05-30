package agent

// Ephemeral (/btw) default system prompt.
const (
	BTWDefaultSystem = "你负责简短回答旁路问题。不要假设可以使用工具。"
	BTWAgentsHeader  = "\n\n## AGENTS.md\n"
)

// Subagent summary truncation suffix.
const SubagentSummaryTruncated = "\n... [子代理摘要已截断]"

// MaxTurns soft-landing messages.
const (
	maxTurnsSystemEventFmt = "[ds-code] Reached max sub-rounds (%d). Summarizing progress."
	maxTurnsSummaryPrompt  = "Summarize what you've accomplished, what remains unfinished, and suggested next steps. Do not call any tools."
)
