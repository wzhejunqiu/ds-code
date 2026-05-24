package session

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/component"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

func TestScheduleResumeFilterSkipsWhenEmptyListLoaded(t *testing.T) {
	s := &state.State{
		Overlay:        state.OverlayResume,
		ResumeFilter:   "nomatch",
		ResumeSessions: []session.Summary{},
	}
	var picker component.Picker

	if cmd := ScheduleResumeFilter(s, "nomatch", &picker); cmd != nil {
		t.Fatal("expected no refetch when filter unchanged and empty list already loaded")
	}
	if s.OverlayText == resumeLoadingOverlay {
		t.Fatal("should not flash loading overlay when list is already loaded")
	}
}

func TestScheduleResumeFilterRefetchesWhenFilterChanges(t *testing.T) {
	s := &state.State{
		Overlay:        state.OverlayResume,
		ResumeFilter:   "foo",
		ResumeSessions: []session.Summary{{ID: "foo-1"}},
	}
	var picker component.Picker

	cmd := ScheduleResumeFilter(s, "bar", &picker)
	if cmd == nil {
		t.Fatal("expected fetch when filter changes")
	}
	if s.ResumeSessions != nil {
		t.Fatal("expected stale sessions cleared while loading new filter")
	}
	if s.OverlayText != resumeLoadingOverlay {
		t.Fatal("expected loading overlay while fetching new filter")
	}
}

func TestApplyResumeSessionsUsesNonNilEmptySlice(t *testing.T) {
	s := &state.State{}
	var picker component.Picker

	ApplyResumeSessions(s, "x", nil, &picker)
	if s.ResumeSessions == nil {
		t.Fatal("expected non-nil empty slice so loaded state is distinguishable from unloaded")
	}
}
