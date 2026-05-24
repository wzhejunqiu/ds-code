//go:build tuitest

package input

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

// TCaseRunner handles /tcase when set by the harness binary.
var TCaseRunner func(s *state.State, args string, syncChat, syncTool func()) tea.Cmd

func trySubmitTUITestSlash(cmd, args string, s *state.State, syncChat, syncTool func()) (tea.Cmd, bool) {
	if cmd != "tcase" {
		return nil, false
	}
	if TCaseRunner == nil {
		return RunSlash(s, "/tcase "+args), true
	}
	return TCaseRunner(s, args, syncChat, syncTool), true
}
