//go:build !debug

package input

import tea "charm.land/bubbletea/v2"

func trySubmitDevSlash(cmd, _ string) (tea.Cmd, bool) {
	return nil, false
}
