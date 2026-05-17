package tui

import (
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/session"
)

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

func TestEnsureResumeScrollVisible(t *testing.T) {
	m := model{height: 30}
	m.resumeSessions = make([]session.Summary, 25)
	for i := range m.resumeSessions {
		m.resumeSessions[i] = session.Summary{ID: "id", Title: "t"}
	}
	page := m.resumePageSize()

	m.resumeIdx = 0
	m.resumeScroll = 0
	m.ensureResumeScrollVisible()
	if m.resumeScroll != 0 {
		t.Fatalf("scroll = %d, want 0", m.resumeScroll)
	}

	m.resumeIdx = 20
	m.resumeScroll = 0
	m.ensureResumeScrollVisible()
	if m.resumeIdx < m.resumeScroll || m.resumeIdx >= m.resumeScroll+page {
		t.Fatalf("idx=%d scroll=%d page=%d, selection not in window", m.resumeIdx, m.resumeScroll, page)
	}
	if m.resumeScroll == 0 {
		t.Fatal("expected scroll offset to advance for selection near end")
	}

	m.resumeIdx = 24
	m.ensureResumeScrollVisible()
	maxScroll := 25 - page
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.resumeScroll != maxScroll {
		t.Fatalf("scroll = %d, want %d", m.resumeScroll, maxScroll)
	}
}

func TestResumePickerDownScrollsWindow(t *testing.T) {
	m := model{height: 30}
	m.resumeSessions = make([]session.Summary, 20)
	m.overlay = overlayResume
	m.resumeIdx = 0
	m.resumeScroll = 0

	page := m.resumePageSize()
	steps := page + 2
	for i := 0; i < steps; i++ {
		m.resumeMoveSelection(1)
	}
	if m.resumeScroll == 0 {
		t.Fatal("expected window to scroll down after many down presses")
	}
	if m.resumeIdx != steps {
		t.Fatalf("idx = %d, want %d", m.resumeIdx, steps)
	}
}
