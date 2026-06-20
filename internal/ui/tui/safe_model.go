package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/wzhejunqiu/ds-code/internal/ui/theme"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/style"
)

// safeModel wraps the TUI model and recovers from render/update panics.
type safeModel struct {
	inner *model.Model
}

func newSafeModel(d *Deps) *safeModel {
	return &safeModel{inner: model.New(d)}
}

func (s *safeModel) Init() tea.Cmd {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.Init()
}

func (s *safeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s == nil || s.inner == nil {
		return s, nil
	}
	var cmd tea.Cmd
	func() {
		defer func() {
			if r := recover(); r != nil {
				s.inner.ErrLine = formatRecoveredError("update", r)
				cmd = nil
			}
		}()
		updated, c := s.inner.Update(msg)
		if m, ok := updated.(*model.Model); ok {
			s.inner = m
		}
		cmd = c
	}()
	return s, cmd
}

func (s *safeModel) View() tea.View {
	if s == nil || s.inner == nil {
		v := tea.NewView(style.App.Render("TUI internal error\n"))
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}
	var view tea.View
	func() {
		defer func() {
			if r := recover(); r != nil {
				s.inner.ErrLine = formatRecoveredError("render", r)
				view = s.fallbackView()
			}
		}()
		view = s.inner.View()
	}()
	if view.Content == "" && !view.AltScreen {
		view = s.fallbackView()
	}
	return view
}

func (s *safeModel) fallbackView() tea.View {
	msg := "TUI render error"
	if s.inner != nil && s.inner.ErrLine != "" {
		msg = s.inner.ErrLine
	}
	v := tea.NewView(style.App.Render(
		lipgloss.NewStyle().Foreground(theme.Error).Render(msg) +
			"\n\nPress Esc to clear this message.",
	))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func formatRecoveredError(phase string, r any) string {
	return fmt.Sprintf("TUI %s recovered: %v", phase, r)
}
