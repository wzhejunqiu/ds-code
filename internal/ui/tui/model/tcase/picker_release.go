//go:build !tuitest

package tcase

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/ui/tui/component"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
)

func UpdatePicker(*state.State, msg.TCasePickerMsg, *component.Picker) tea.Cmd {
	return nil
}

func HandleKey(*state.State, *component.Picker, tea.KeyMsg) bool {
	return false
}

func ConfirmSelection(*state.State, *component.Picker, func(), func()) (tea.Cmd, bool) {
	return nil, false
}

func ClearPicker(*state.State, *component.Picker) {}

func SyncPicker(*state.State, *component.Picker) {}
