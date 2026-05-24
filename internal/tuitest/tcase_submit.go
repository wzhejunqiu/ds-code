//go:build tuitest

package tuitest

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/tuitest/mockserver"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/input"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

// NewTCaseSubmit returns a slash handler that runs built-in scenarios.
func NewTCaseSubmit(reg *mockserver.Registry) func(s *state.State, args string, syncChat, syncTool func()) tea.Cmd {
	return func(s *state.State, args string, syncChat, syncTool func()) tea.Cmd {
		args = strings.TrimSpace(args)
		if args == "" || args == "list" {
			return listScenarios(reg)
		}
		parts := strings.Fields(args)
		if len(parts) >= 2 && parts[0] == "run" {
			name := parts[1]
			if err := reg.SetActive(name); err != nil {
				return func() tea.Msg { return msg.TurnDoneMsg{Err: err} }
			}
			sc, ok := reg.Get(name)
			if !ok {
				return func() tea.Msg { return msg.TurnDoneMsg{Err: fmt.Errorf("unknown scenario %q", name)} }
			}
			return input.SubmitLine(s, sc.Prompt, syncChat, syncTool)
		}
		return func() tea.Msg {
			return msg.SlashOutputMsg{Text: "usage: /tcase [list|run <name>]"}
		}
	}
}

func listScenarios(reg *mockserver.Registry) tea.Cmd {
	return func() tea.Msg {
		var items []msg.TCasePickerItem
		for _, name := range reg.List() {
			if sc, ok := reg.Get(name); ok {
				items = append(items, msg.TCasePickerItem{Name: name, Desc: sc.Prompt})
			}
		}
		return msg.TCasePickerMsg{Items: items}
	}
}
