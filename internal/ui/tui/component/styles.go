// List row styles for Picker.View (selected vs normal).
package component

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/wzhejunqiu/ds-code/internal/ui/theme"
)

var (
	styleItem         = lipgloss.NewStyle().Foreground(theme.Text)
	styleItemSelected = lipgloss.NewStyle().
				Background(theme.UserBg).
				Foreground(theme.Text).
				Bold(true)
)
