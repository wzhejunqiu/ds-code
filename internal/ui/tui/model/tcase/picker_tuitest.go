//go:build tuitest

package tcase

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/ui/tui/component"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/input"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/session"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
)

var tcasePickerKeys = component.PickerKeyOpts{Tab: component.PickerTabMoveDown}

func UpdatePicker(s *state.State, m msg.TCasePickerMsg, picker *component.Picker) tea.Cmd {
	items := make([]state.TCaseItem, len(m.Items))
	for i, it := range m.Items {
		items[i] = state.TCaseItem{Name: it.Name, Desc: it.Desc}
	}
	s.TCaseItems = items
	s.Overlay = state.OverlayTCase
	s.Complete = nil
	s.CompleteFilterKey = ""
	picker.ResetSelection()
	SyncPicker(s, picker)
	return nil
}

// SyncPicker rebuilds the /tcase overlay after resize or navigation.
func SyncPicker(s *state.State, picker *component.Picker) {
	rows := make([]string, len(s.TCaseItems))
	for i, it := range s.TCaseItems {
		rows[i] = fmt.Sprintf("%-18s %s", it.Name, it.Desc)
	}
	picker.Header = "TUI integration scenarios (↑↓ select, Enter run, Esc dismiss):"
	picker.Empty = "No scenarios."
	picker.PageSize = session.PageSize(s)
	picker.SetItems(rows)
	s.OverlayText = picker.View()
}

func HandleKey(s *state.State, picker *component.Picker, msg tea.KeyMsg) bool {
	if s.Overlay != state.OverlayTCase {
		return false
	}
	action, handled := picker.HandleKey(msg, tcasePickerKeys)
	if !handled {
		return false
	}
	if action == component.PickerKeyCancel {
		ClearPicker(s, picker)
		return true
	}
	SyncPicker(s, picker)
	return true
}

func ConfirmSelection(s *state.State, picker *component.Picker, syncChat, syncTool func()) (tea.Cmd, bool) {
	if s.Overlay != state.OverlayTCase || len(s.TCaseItems) == 0 {
		return nil, false
	}
	idx := picker.Cursor
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s.TCaseItems) {
		idx = len(s.TCaseItems) - 1
	}
	name := s.TCaseItems[idx].Name
	ClearPicker(s, picker)
	if input.TCaseRunner == nil {
		return nil, true
	}
	return input.TCaseRunner(s, "run "+name, syncChat, syncTool), true
}

func ClearPicker(s *state.State, picker *component.Picker) {
	s.Overlay = state.OverlayNone
	s.TCaseItems = nil
	picker.Clear()
	s.OverlayText = ""
}
