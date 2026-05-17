package tui

import (
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/session"
)

func TestResumePageSizeBounds(t *testing.T) {
	m := model{height: 100}
	if got := m.resumePageSize(); got != 14 {
		t.Fatalf("height=100: pageSize = %d, want 14", got)
	}
	m.height = 20
	if got := m.resumePageSize(); got != 4 {
		t.Fatalf("height=20: pageSize = %d, want 4 (minimum)", got)
	}
	m.height = 0
	if got := m.resumePageSize(); got != 8 {
		t.Fatalf("height=0: pageSize = %d, want 8 (default)", got)
	}
}

func TestUpdateResumePickerSkipsWhenFilterUnchanged(t *testing.T) {
	m := model{}
	m.overlay = overlayResume
	m.resumeFilter = "foo"
	m.resumeSessions = []session.Summary{{ID: "aaa", Title: "first"}}
	m.resumePicker.Cursor = 0
	m.overlayText = "cached overlay"

	m.updateResumePicker("foo")
	if m.overlayText != "cached overlay" {
		t.Fatalf("overlay re-rendered on cursor blink: %q", m.overlayText)
	}
}

func TestResumePickerScrollWithPageSize(t *testing.T) {
	m := model{height: 30}
	m.resumeSessions = make([]session.Summary, 25)
	for i := range m.resumeSessions {
		m.resumeSessions[i] = session.Summary{ID: "id", Title: "t"}
	}
	m.overlay = overlayResume
	m.syncResumePicker()

	page := m.resumePageSize()
	steps := page + 2
	for i := 0; i < steps; i++ {
		m.resumePicker.Move(1)
	}
	if m.resumePicker.Scroll == 0 {
		t.Fatal("expected window to scroll down after many down presses")
	}
	if m.resumePicker.Cursor != steps {
		t.Fatalf("cursor = %d, want %d", m.resumePicker.Cursor, steps)
	}
}

func TestFormatResumeUpdated(t *testing.T) {
	if got := formatResumeUpdated(time.Time{}); got != "-" {
		t.Fatalf("zero time = %q", got)
	}
	ts := time.Date(2025, 5, 17, 14, 30, 0, 0, time.UTC)
	got := formatResumeUpdated(ts)
	if got != ts.Local().Format("2006-01-02 15:04") {
		t.Fatalf("got %q", got)
	}
}
