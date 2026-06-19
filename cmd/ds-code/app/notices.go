package app

import (
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/header"
)

func buildStartupNotices(a *App) []header.Notice {
	var notices []header.Notice
	if a.Cfg != nil && a.Cfg.AllowLogSensitiveData {
		notices = append(notices, header.Notice{
			Level: header.NoticeWarn,
			Text:  logging.SensitiveDataWarningMsg,
		})
	}
	if a.mcpMgr != nil {
		if skipped := a.mcpMgr.SkippedTools(); len(skipped) > 0 {
			notices = append(notices, header.Notice{
				Level: header.NoticeWarn,
				Text:  header.FormatMCPSkippedSummary(skipped),
			})
		}
	}
	return notices
}
