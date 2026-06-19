package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/component"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/history"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

const (
	resumeListMax        = 50
	resumeFilterDebounce = 150 * time.Millisecond
	resumeLoadingOverlay = "Loading sessions…"
)

var resumePickerKeys = component.PickerKeyOpts{Tab: component.PickerTabMoveDown}

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

func SyncResumePicker(s *state.State, picker *component.Picker) {
	items := make([]string, len(s.ResumeSessions))
	for i, sum := range s.ResumeSessions {
		title := session.OneLine(sum.Title)
		if title == "" {
			title = "(untitled)"
		}
		items[i] = fmt.Sprintf("%s  %s  %s", sum.ID, title, formatUpdated(sum.UpdatedAt))
	}
	picker.Header = "Recent sessions (↑↓ scroll, PgUp/PgDn, Enter to resume):"
	picker.Empty = "No matching sessions."
	picker.PageSize = PageSize(s)
	picker.SetItems(items)
	s.OverlayText = picker.View()
}

func filterSessions(list []session.Summary, filter string) []session.Summary {
	if filter == "" {
		return list
	}
	var filtered []session.Summary
	lower := strings.ToLower(filter)
	for _, sum := range list {
		if strings.HasPrefix(sum.ID, filter) ||
			strings.Contains(strings.ToLower(sum.ID), lower) ||
			strings.Contains(strings.ToLower(sum.Title), lower) {
			filtered = append(filtered, sum)
		}
	}
	return filtered
}

func listFromStore(store session.Store, filter string) ([]session.Summary, error) {
	list, err := store.ListSessions(context.Background(), resumeListMax)
	if err != nil {
		return nil, err
	}
	return filterSessions(list, filter), nil
}

func formatUpdated(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04")
}

func HandleResumeKey(s *state.State, msg tea.KeyMsg, picker *component.Picker) bool {
	action, handled := picker.HandleKey(msg, resumePickerKeys)
	if !handled {
		return false
	}
	if action == component.PickerKeyCancel {
		ClearResumePicker(s, picker)
		return true
	}
	SyncResumePicker(s, picker)
	return true
}

func ClearResumePicker(s *state.State, picker *component.Picker) {
	s.Overlay = state.OverlayNone
	s.ResumeSessions = nil
	s.ResumeFilter = ""
	picker.Clear()
	s.OverlayText = ""
	s.ResumePending = false
}

func ScheduleResumeFilter(s *state.State, filter string, picker *component.Picker) tea.Cmd {
	// ResumeSessions == nil means not loaded yet; non-nil (including empty) means loaded.
	if s.Overlay == state.OverlayResume && filter == s.ResumeFilter && s.ResumeSessions != nil {
		return nil
	}
	s.Overlay = state.OverlayResume
	s.Complete = nil
	s.CompleteFilterKey = ""
	if filter != s.ResumeFilter {
		s.ResumeSessions = nil
	}
	if s.ResumeSessions == nil {
		s.OverlayText = resumeLoadingOverlay
	}
	s.ResumeFilterSeq++
	seq := s.ResumeFilterSeq
	return tea.Tick(resumeFilterDebounce, func(time.Time) tea.Msg {
		return msg.ResumeFilterTickMsg{Filter: filter, Seq: seq}
	})
}

func UpdateResumeFilterTick(s *state.State, m msg.ResumeFilterTickMsg) tea.Cmd {
	if m.Seq != s.ResumeFilterSeq {
		return nil
	}
	return FetchSessions(s, m.Filter, m.Seq)
}

func ApplyResumeSessions(s *state.State, filter string, list []session.Summary, picker *component.Picker) {
	filterChanged := s.ResumeFilter != filter
	s.ResumeFilter = filter
	if list == nil {
		list = []session.Summary{}
	}
	s.ResumeSessions = list
	if filterChanged || len(list) == 0 {
		picker.ResetSelection()
	} else {
		picker.ClampSelection()
	}
	s.Overlay = state.OverlayResume
	SyncResumePicker(s, picker)
}

func FetchSessions(s *state.State, filter string, seq uint64) tea.Cmd {
	d := s.Deps
	return func() tea.Msg {
		list, err := listFromStore(d.Store, filter)
		return msg.ResumeListMsg{Filter: filter, Seq: seq, Sessions: list, Err: err}
	}
}

func FetchList(s *state.State) tea.Cmd {
	s.ResumeFilterSeq++
	seq := s.ResumeFilterSeq
	return FetchSessions(s, "", seq)
}

func LoadInitialHistory(s *state.State) tea.Cmd {
	if s.Deps == nil || s.Deps.Store == nil || s.SessionID == "" {
		return nil
	}
	d := s.Deps
	sid := s.SessionID
	reasoningOpen := s.ReasoningAll
	return func() tea.Msg {
		var tools *tool.Registry
		if d.Runner != nil {
			tools = d.Runner.Tools
		}
		chat, err := history.LoadSessionChat(d.Store, sid, reasoningOpen, d.Cfg.ProjectRoot, tools)
		if err != nil {
			return msg.HistoryLoadedMsg{Err: err}
		}
		reg, err := history.LoadSubagentRegistry(context.Background(), d.Subagent, sid, reasoningOpen, d.Cfg.ProjectRoot, tools)
		return msg.HistoryLoadedMsg{Chat: chat, Subagents: reg, Err: err}
	}
}

func ResumeSession(s *state.State, id string) tea.Cmd {
	d := s.Deps
	reasoningOpen := s.ReasoningAll
	return func() tea.Msg {
		ctx := context.Background()
		if _, err := d.Store.Get(ctx, id); err != nil {
			return msg.SessionResumedMsg{Err: err}
		}
		var tools *tool.Registry
		if d.Runner != nil {
			tools = d.Runner.Tools
		}
		chat, err := history.LoadSessionChat(d.Store, id, reasoningOpen, d.Cfg.ProjectRoot, tools)
		if err != nil {
			return msg.SessionResumedMsg{Err: err}
		}
		reg, err := history.LoadSubagentRegistry(ctx, d.Subagent, id, reasoningOpen, d.Cfg.ProjectRoot, tools)
		if err != nil {
			return msg.SessionResumedMsg{Err: err}
		}
		return msg.SessionResumedMsg{SessionID: id, Chat: chat, Subagents: reg}
	}
}
