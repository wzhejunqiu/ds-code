package diagnostics

const (
	DescDiagnostics = "对文件或目录运行语言服务器诊断（gopls、tsserver、clangd 等）。"

	SchemaSeverity = "筛选：error、warning、info、hint"

	ErrLSPDisabled          = "配置中已禁用 LSP"
	ResultNoDiagnosticFiles = "未找到可诊断的文件。"
	ResultNoDiagnostics     = "无诊断信息。"
	NoteNoDiagnosticFiles   = "\n未找到可诊断的文件。"
	NoteSkipNoServer        = "--- 跳过 %s：无对应 LSP 服务器"
	NotePathError           = "--- %s: %v"
	NoteRelPathError        = "--- %s: %v"
	ResultNoIssues          = "%s: （无问题）"
)
