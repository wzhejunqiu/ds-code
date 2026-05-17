package component

import "github.com/charmbracelet/lipgloss"

var (
	colorText   = lipgloss.Color("#3D3D3D")
	colorUserBg = lipgloss.Color("#E8E6E0")

	styleItem         = lipgloss.NewStyle().Foreground(colorText)
	styleItemSelected = lipgloss.NewStyle().
				Background(colorUserBg).
				Foreground(colorText).
				Bold(true)
)
