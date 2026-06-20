package tui

import (
	tea "charm.land/bubbletea/v2"
)

// Run starts the interactive Bubble Tea UI.
func Run(deps Deps) error {
	events := make(chan tea.Msg, 256)
	deps.Events = events
	d := deps
	m := newSafeModel(&d)
	p := tea.NewProgram(m, tea.WithoutCatchPanics())
	go func() {
		for msg := range events {
			p.Send(msg)
		}
	}()
	_, err := p.Run()
	close(events)
	return err
}
