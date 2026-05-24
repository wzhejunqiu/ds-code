package subagent

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/component"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/subagent"
)

var listPickerKeys = component.PickerKeyOpts{Tab: component.PickerTabMoveDown}

func PageSize(s *state.State) int {
	if s.Height <= 0 {
		return 8
	}
	n := s.Height/5 - 1
	if n < 4 {
		n = 4
	}
	if n > 14 {
		n = 14
	}
	return n
}

func OpenList(s *state.State, picker *component.Picker) tea.Cmd {
	if s.Subagents.Len() == 0 {
		return nil
	}
	s.ViewingSubagentID = ""
	s.SubagentNav = state.SubagentNavList
	s.Overlay = state.OverlaySubagentList
	s.SyncDisplayedChat()
	picker.ResetSelection()
	SyncListPicker(s, picker)
	return nil
}

func SyncListPicker(s *state.State, picker *component.Picker) {
	items := make([]string, 0, s.Subagents.Len())
	for _, rec := range s.Subagents.All() {
		items = append(items, formatListItem(rec))
	}
	picker.Header = "Subagents (↑↓ select, Enter view, ← back, Esc dismiss):"
	picker.Empty = "No subagents yet."
	picker.PageSize = PageSize(s)
	picker.SetItems(items)
	s.OverlayText = picker.View()
}

func formatListItem(rec *subagent.Record) string {
	if rec == nil {
		return ""
	}
	label := rec.Label
	if label == "" {
		label = rec.ID
	}
	status := "running"
	switch rec.Status {
	case subagent.StatusDone:
		status = "done"
	case subagent.StatusError:
		status = "error"
	}
	elapsed := ""
	if !rec.StartedAt.IsZero() {
		end := rec.EndedAt
		if end.IsZero() {
			end = time.Now()
		}
		elapsed = fmt.Sprintf(" · %s", formatDuration(end.Sub(rec.StartedAt)))
	}
	return fmt.Sprintf("%s  %s  [%s%s]", rec.ID, truncate(label, 40), status, elapsed)
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Round(time.Second).Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Round(time.Second).Seconds()) % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm%ds", m, s)
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func OpenDetail(s *state.State, id string, picker *component.Picker, sync func()) {
	rec := s.Subagents.Get(id)
	if rec == nil {
		return
	}
	s.ViewingSubagentID = id
	s.SubagentNav = state.SubagentNavDetail
	s.Overlay = state.OverlayNone
	s.OverlayText = ""
	picker.Clear()
	s.SyncDisplayedChat()
	sync()
}

func Back(s *state.State, picker *component.Picker, sync func()) bool {
	switch s.SubagentNav {
	case state.SubagentNavDetail:
		s.SubagentNav = state.SubagentNavList
		s.ViewingSubagentID = ""
		s.SyncDisplayedChat()
		sync()
		if s.Subagents.Len() > 0 {
			s.Overlay = state.OverlaySubagentList
			SyncListPicker(s, picker)
		} else {
			s.SubagentNav = state.SubagentNavMain
		}
		return true
	case state.SubagentNavList:
		s.Overlay = state.OverlayNone
		s.OverlayText = ""
		picker.Clear()
		s.SubagentNav = state.SubagentNavMain
		s.SyncDisplayedChat()
		sync()
		return true
	default:
		return false
	}
}

func HandleListKey(s *state.State, msg tea.KeyMsg, picker *component.Picker, sync func()) (tea.Cmd, bool) {
	if s.SubagentNav != state.SubagentNavList {
		return nil, false
	}
	action, handled := picker.HandleKey(msg, listPickerKeys)
	if !handled {
		if msg.String() == "left" {
			Back(s, picker, sync)
			return nil, true
		}
		return nil, false
	}
	switch action {
	case component.PickerKeyCancel:
		Back(s, picker, sync)
	case component.PickerKeyConfirm:
		records := s.Subagents.All()
		if picker.Cursor >= 0 && picker.Cursor < len(records) {
			OpenDetail(s, records[picker.Cursor].ID, picker, sync)
		}
	default:
		SyncListPicker(s, picker)
	}
	return nil, true
}

func HandleNavKey(s *state.State, msg tea.KeyMsg, picker *component.Picker, sync func()) (tea.Cmd, bool) {
	overlayBlocks := s.Overlay != state.OverlayNone &&
		s.Overlay != state.OverlaySubagentList
	switch msg.String() {
	case "down":
		if overlayBlocks {
			break
		}
		if s.SubagentNav == state.SubagentNavDetail && s.Subagents.Len() > 0 {
			return OpenList(s, picker), true
		}
		if s.SubagentNav == state.SubagentNavMain && s.Subagents.Len() > 0 {
			return OpenList(s, picker), true
		}
	case "left":
		if overlayBlocks && s.SubagentNav == state.SubagentNavMain {
			break
		}
		if Back(s, picker, sync) {
			return nil, true
		}
	case "esc":
		if s.SubagentNav == state.SubagentNavList {
			Back(s, picker, sync)
			return nil, true
		}
		if s.SubagentNav == state.SubagentNavDetail {
			Back(s, picker, sync)
			return nil, true
		}
	}
	if s.SubagentNav == state.SubagentNavList {
		return HandleListKey(s, msg, picker, sync)
	}
	return nil, false
}

// DetailBreadcrumb returns a short path label for the detail view header.
func DetailBreadcrumb(s *state.State) string {
	rec := s.ViewingSubagent()
	if rec == nil {
		return ""
	}
	label := rec.Label
	if label == "" {
		label = rec.ID
	}
	return fmt.Sprintf("← subagents / %s", truncate(label, 48))
}
