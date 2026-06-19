package model

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

func (m *Model) viewportHPEnabled() bool {
	if m.selDragging || m.selRange.Active() {
		return false
	}
	if m.Overlay != state.OverlayNone || m.Prompt != nil {
		return false
	}
	return true
}

func (m *Model) applyViewportHP() {
	enabled := m.viewportHPEnabled()
	m.chatVP.HighPerformanceRendering = enabled
	m.toolVP.HighPerformanceRendering = enabled && m.ToolOpen && m.toolVP.Height > 0
}

func (m *Model) viewportScrollCmdFromLines(vp *viewport.Model, lines []string, scrollDown bool) tea.Cmd {
	if !m.viewportHPEnabled() || len(lines) == 0 {
		return nil
	}
	if scrollDown {
		return viewport.ViewDown(*vp, lines)
	}
	return viewport.ViewUp(*vp, lines)
}

func (m *Model) viewportSyncCmdFor(vp *viewport.Model) tea.Cmd {
	if !m.viewportHPEnabled() || vp == nil || vp.Height <= 0 {
		return nil
	}
	return viewport.Sync(*vp)
}

func (m *Model) viewportSyncCmd() tea.Cmd {
	if !m.viewportHPEnabled() {
		return nil
	}
	cmds := []tea.Cmd{m.viewportSyncCmdFor(&m.chatVP)}
	if m.ToolOpen && m.toolVP.Height > 0 {
		cmds = append(cmds, m.viewportSyncCmdFor(&m.toolVP))
	}
	return tea.Batch(cmds...)
}

// syncChatViewportHP returns viewport.Sync after synchronous SetContent updates.
// Required for HighPerformanceRendering; call from Update handlers that invoke syncChatView directly.
func (m *Model) syncChatViewportHP() tea.Cmd {
	m.applyViewportHP()
	return m.viewportSyncCmd()
}

func (m *Model) withHPSync(cmd tea.Cmd) tea.Cmd {
	hp := m.syncChatViewportHP()
	if hp == nil {
		return cmd
	}
	if cmd == nil {
		return hp
	}
	return tea.Batch(cmd, hp)
}
