package logging

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
)

// SensitiveDataWarningMsg is shown when --allow-log-sensitive-data is active (requires -vv).
const SensitiveDataWarningMsg = "警告：已启用 --allow-log-sensitive-data。调试日志可能包含对话内容、文件路径等敏感信息，请勿分享 ds-code.log。"

var sensitiveWarnStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#DC2626")).
	Bold(true)

// PrintSensitiveDataWarning writes the sensitive-logging warning to w in red.
func PrintSensitiveDataWarning(w io.Writer) {
	fmt.Fprintln(w, sensitiveWarnStyle.Render(SensitiveDataWarningMsg))
}
