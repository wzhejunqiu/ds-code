package component

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/hejunqiu/ds-code/internal/ui/theme"
)

var (
	styleItem = lipgloss.NewStyle().Foreground(theme.Text)
	styleItemSelected = lipgloss.NewStyle().
				Background(theme.UserBg).
				Foreground(theme.Text).
				Bold(true)
)
