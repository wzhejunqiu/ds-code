package model

import (
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/overlay"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/tcase"
)

func (m *Model) dismissOverlay() {
	overlay.Dismiss(&m.State)
	m.completePicker.Clear()
	m.resumePicker.Clear()
	tcase.ClearPicker(&m.State, &m.tcasePicker)
}
