package tui

import (
	"testing"
	"time"
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
