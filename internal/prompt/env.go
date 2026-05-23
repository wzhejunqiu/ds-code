package prompt

import (
	"strings"
	"time"
)

// FormatRuntimeEnv builds the body of the runtime environment section (no section header).
func FormatRuntimeEnv(projectRoot, cwd string, now time.Time, platformLines []string) string {
	var lines []string
	if s := strings.TrimSpace(projectRoot); s != "" {
		lines = append(lines, "工作区（project_root）："+s)
	}
	if s := strings.TrimSpace(cwd); s != "" {
		lines = append(lines, "当前目录（cwd）："+s)
	}
	for _, pl := range platformLines {
		if pl = strings.TrimSpace(pl); pl != "" {
			lines = append(lines, pl)
		}
	}
	if !now.IsZero() {
		lines = append(lines, "当前时间："+now.Format("2006-01-02 15:04:05 MST"))
		lines = append(lines, "今日日期："+now.Format("Monday, 2006-01-02"))
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}
