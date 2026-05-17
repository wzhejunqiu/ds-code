package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hejunqiu/ds-code/internal/billing"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/session"
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

func renderHeader(width int, version string, cfg *config.Config, sess *session.Session) string {
	if width < 20 {
		width = 20
	}

	title := fmt.Sprintf("ds-code v%s", version)
	titleLine := styleHeaderTitle.Render(title)

	var metaLine string
	if sess != nil {
		thinking := sess.ThinkingType
		if thinking == "" {
			thinking = cfg.LLM.Thinking.Type
		}
		effort := sess.ReasoningEffort
		if effort == "" {
			effort = cfg.LLM.ReasoningEffort
		}
		modelLabel := sess.Model
		if effort != "" {
			modelLabel += "[" + effort + "]"
		}
		snap := session.UsageSnapshotFromSession(*sess)
		cost := billing.FormatUSD(billing.EstimateUSD(sess.Model, snap))
		metaLine = styleHeaderMeta.Render(fmt.Sprintf("%s · %s · %s", modelLabel, thinkingLabel(thinking), cost))
	} else {
		metaLine = styleHeaderMeta.Render(cfg.LLM.Model + " · API usage estimate")
	}

	pathLine := styleHeaderPath.Render(shortenHome(cfg.ProjectRoot))

	logo := styleLogo.Render(logoArt)
	info := lipgloss.JoinVertical(lipgloss.Left, titleLine, metaLine, pathLine)
	return lipgloss.JoinHorizontal(lipgloss.Center, logo, "  ", info)
}

func thinkingLabel(t string) string {
	if t == "" {
		return "standard"
	}
	return t
}

func renderDivider(width int) string {
	if width < 1 {
		width = 1
	}
	return styleDivider.Render(strings.Repeat("─", width))
}

func renderInputFrame(width int, inputView string) string {
	innerW := width - 4
	if innerW < 10 {
		innerW = 10
	}
	prompt := styleInputPrompt.Render("> ")
	line := lipgloss.JoinHorizontal(lipgloss.Top, prompt, styleInputText.Render(inputView))
	var b strings.Builder
	b.WriteString(renderDivider(width))
	b.WriteString("\n")
	b.WriteString(line)
	b.WriteString("\n")
	b.WriteString(renderDivider(width))
	return b.String()
}

func renderFooter(width int, left, right string) string {
	if width < 20 {
		width = 20
	}
	leftStyled := styleFooterHint.Render(left)
	rightStyled := styleFooterStat.Render(right)
	gap := width - lipgloss.Width(leftStyled) - lipgloss.Width(rightStyled)
	if gap < 1 {
		gap = 1
	}
	return leftStyled + strings.Repeat(" ", gap) + rightStyled
}
