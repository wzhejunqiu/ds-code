//go:build !tuitest

package input

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

func trySubmitTUITestSlash(_ string, _ string, _ *state.State, _, _ func()) (tea.Cmd, bool) {
	return nil, false
}
