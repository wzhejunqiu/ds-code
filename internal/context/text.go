package context

// Compact summarization prompts.
const (
	CompactSummarizeSystem     = "你负责生成简洁的技术对话摘要。"
	CompactSummarizeUserPrefix = `请摘要以下对话以便继续工作。保留：目标、决策、文件路径、错误与未完成任务。不要包含密钥或凭证。力求简洁。

`
	CompactFallbackSummary = "[较早回合已从 API 上下文中省略；联网后可执行 /compact 生成完整摘要。]"
)

// Conversation summary message prefix.
const ConversationSummaryPrefix = "[对话摘要]\n"

// formatTurnsForCompact labels.
const (
	CompactTurnLabel           = "--- 回合 %d ---\n"
	CompactRoleUser            = "用户: "
	CompactRoleAssistant       = "助手: "
	CompactRoleAssistantReason = "助手（推理）: "
	CompactRoleTool            = "工具 "
	CompactTruncated           = "\n...[已截断]"
	CompactRedacted            = "[已脱敏]"
)

// @ reference expansion messages.
const (
	AtRefSkippedBudget    = "[已跳过：@ 引用预算已用尽]"
	AtRefErrorLine        = "错误: %v"
	AtRefFileTruncated    = "\n... [@ 引用预算导致文件截断]"
	AtRefRemainingSkipped = "\n... [其余文件已跳过：预算用尽]"
	AtRefTooManyFiles     = "文件过多（%d+）。仅列出前 %d 个；更多请用 grep/glob。\n\n"
	AtRefDirHeader        = "--- @%s/（目录） ---\n"
	AtRefDirListingFooter = "\n如需文件内容，请使用 read_file 或 glob 按需读取。"
	AtRefFileHeader       = "--- @%s (%s) ---\n"
	AtRefSkippedBlock     = "--- @%s ---\n%s"
)

// Git snapshot truncation.
const GitSnapshotTruncated = "\n... [git 快照已截断]"
