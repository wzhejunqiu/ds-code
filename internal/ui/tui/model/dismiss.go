package model

import "github.com/hejunqiu/ds-code/internal/ui/tui/model/overlay"

func (m *Model) dismissOverlay() {
	overlay.Dismiss(&m.State)
	m.completePicker.Clear()
	m.resumePicker.Clear()
}
