package header

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hejunqiu/ds-code/internal/billing"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/ui/tui/style"
)

func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+string(os.PathSeparator)) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

func Render(width int, version string, cfg *config.Config, sess *session.Session) string {
	if width < 20 {
		width = 20
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	title := fmt.Sprintf("ds-code v%s", version)
	titleLine := style.HeaderTitle.Render(title)

	var metaLine string
	if sess != nil {
		thinking := sess.ThinkingType
		if thinking == "" && cfg.LLM.Thinking.Type != "" {
			thinking = cfg.LLM.Thinking.Type
		}
		effort := sess.ReasoningEffort
		if effort == "" && cfg.LLM.ReasoningEffort != "" {
			effort = cfg.LLM.ReasoningEffort
		}
		modelLabel := sess.Model
		if effort != "" {
			modelLabel += "[" + effort + "]"
		}
		snap := session.UsageSnapshotFromSession(*sess)
		cost := billing.FormatUSD(billing.EstimateUSD(sess.Model, snap))
		metaLine = style.HeaderMeta.Render(fmt.Sprintf("%s · %s · %s", modelLabel, thinkingLabel(thinking), cost))
	} else {
		metaLine = style.HeaderMeta.Render(cfg.LLM.Model + " · API usage estimate")
	}

	pathLine := style.HeaderPath.Render(shortenHome(cfg.ProjectRoot))
	logo := style.Logo.Render(style.LogoArt)
	info := lipgloss.JoinVertical(lipgloss.Left, titleLine, metaLine, pathLine)
	return lipgloss.JoinHorizontal(lipgloss.Center, logo, "  ", info)
}

func thinkingLabel(t string) string {
	if t == "" {
		return "standard"
	}
	return t
}
