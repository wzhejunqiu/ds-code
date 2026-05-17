package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hejunqiu/ds-code/internal/ui/theme"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model"
	"github.com/hejunqiu/ds-code/internal/ui/tui/style"
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
				s.inner.State.ErrLine = formatRecoveredError("update", r)
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

func (s *safeModel) View() string {
	if s == nil || s.inner == nil {
		return style.App.Render("TUI internal error\n")
	}
	var view string
	func() {
		defer func() {
			if r := recover(); r != nil {
				s.inner.State.ErrLine = formatRecoveredError("render", r)
				view = s.fallbackView()
			}
		}()
		view = s.inner.View()
	}()
	if view == "" {
		view = s.fallbackView()
	}
	return view
}

func (s *safeModel) fallbackView() string {
	msg := "TUI render error"
	if s.inner != nil && s.inner.State.ErrLine != "" {
		msg = s.inner.State.ErrLine
	}
	return style.App.Render(
		lipgloss.NewStyle().Foreground(theme.Error).Render(msg) +
			"\n\nPress Esc to clear this message.",
	)
}

func formatRecoveredError(phase string, r any) string {
	return fmt.Sprintf("TUI %s recovered: %v", phase, r)
}
