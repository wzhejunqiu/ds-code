package session

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/component"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
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

func TestClearResumePicker_invalidatesInFlightSeq(t *testing.T) {
	s := &state.State{
		Overlay:         state.OverlayResume,
		ResumeFilterSeq: 1,
	}
	var picker component.Picker

	ClearResumePicker(s, &picker)
	if s.ResumeFilterSeq != 2 {
		t.Fatalf("ResumeFilterSeq = %d, want 2", s.ResumeFilterSeq)
	}

	cmd := UpdateResumeList(s, msg.ResumeListMsg{
		Filter:   "",
		Seq:      1,
		Sessions: []session.Summary{{ID: "stale"}},
	}, &picker)
	if cmd != nil {
		t.Fatal("expected stale ResumeListMsg to be ignored")
	}
	if s.Overlay != state.OverlayNone {
		t.Fatalf("overlay = %v, want none", s.Overlay)
	}
}

func TestUpdateResumeList_ignoredAfterDismiss(t *testing.T) {
	s := &state.State{
		Overlay:         state.OverlayNone,
		ResumePending:   false,
		ResumeFilterSeq: 5,
	}
	var picker component.Picker

	cmd := UpdateResumeList(s, msg.ResumeListMsg{
		Filter:   "",
		Seq:      5,
		Sessions: []session.Summary{{ID: "late"}},
	}, &picker)
	if cmd != nil {
		t.Fatal("expected late ResumeListMsg to be ignored after dismiss")
	}
	if s.Overlay != state.OverlayNone {
		t.Fatalf("overlay = %v, want none", s.Overlay)
	}
}
