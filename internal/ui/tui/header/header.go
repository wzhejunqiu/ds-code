package header

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/style"
	appver "github.com/wzhejunqiu/ds-code/internal/version"
)

const narrowWidth = 72

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

// Render draws the TUI header. scrollOffset scrolls the notification zone when content overflows.
func Render(width int, version string, cfg *config.Config, sess *session.Session, costCNY float64, notices []Notice, scrollOffset int) string {
	if width < 20 {
		width = 20
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	title := fmt.Sprintf("%s v%s", appver.Name, version)
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
		costPart := ""
		if costCNY > 0 {
			costPart = " · " + billing.FormatCNY(costCNY)
		}
		metaLine = style.HeaderMeta.Render(fmt.Sprintf("%s · %s%s", modelLabel, thinkingLabel(thinking), costPart))
	} else {
		metaLine = style.HeaderMeta.Render(cfg.LLM.Model + " · API usage estimate")
	}

	pathLine := style.HeaderPath.Render(shortenHome(cfg.ProjectRoot))
	logo := style.Logo.Render(style.LogoArt)
	info := lipgloss.JoinVertical(lipgloss.Left, titleLine, metaLine, pathLine)
	left := lipgloss.JoinHorizontal(lipgloss.Center, logo, "  ", info)

	if len(notices) == 0 {
		return lipgloss.NewStyle().MaxWidth(width).Render(left)
	}

	narrow := width < narrowWidth
	zoneW := ZoneWidth(width, narrow)
	noticeText := renderNotificationZone(notices, zoneW, scrollOffset)

	if narrow {
		noticeBlock := lipgloss.NewStyle().Width(zoneW).Align(lipgloss.Left).Render(noticeText)
		combined := lipgloss.JoinVertical(lipgloss.Left, info, "", noticeBlock)
		out := lipgloss.JoinHorizontal(lipgloss.Top, logo, "  ", combined)
		return lipgloss.NewStyle().MaxWidth(width).Render(out)
	}

	right := alignNotificationColumn(noticeText, zoneW, width, lipgloss.Width(left))
	out := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
	return lipgloss.NewStyle().MaxWidth(width).Render(out)
}

// alignNotificationColumn places the notification block on the right side of the header
// while keeping wrapped lines left-aligned within the zone (avoid per-line right-align jagged edges).
func alignNotificationColumn(noticeText string, zoneW, headerWidth, leftWidth int) string {
	content := lipgloss.NewStyle().Width(zoneW).Align(lipgloss.Left).Render(noticeText)
	gap := 1
	remain := headerWidth - leftWidth - gap
	if remain < zoneW {
		remain = zoneW
	}
	return lipgloss.NewStyle().Width(remain).Align(lipgloss.Right).Render(content)
}

func thinkingLabel(t string) string {
	if t == "" {
		return "standard"
	}
	return t
}

// NoticesFingerprint returns a stable cache key fragment for notices.
func NoticesFingerprint(notices []Notice) string {
	var sb strings.Builder
	for _, n := range notices {
		fmt.Fprintf(&sb, "%d:%s|", n.Level, n.Text)
	}
	return sb.String()
}
