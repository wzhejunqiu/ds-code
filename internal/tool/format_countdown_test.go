package tool

import (
	"testing"
	"time"
)

func TestFormatTimeoutCountdown(t *testing.T) {
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		deadline time.Time
		want     string
	}{
		{"90s remaining", now.Add(90 * time.Second), "1:30"},
		{"45s remaining", now.Add(45 * time.Second), "45s"},
		{"expired", now.Add(-time.Second), "0s"},
		{"zero deadline", time.Time{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatTimeoutCountdown(tt.deadline, now); got != tt.want {
				t.Fatalf("FormatTimeoutCountdown() = %q, want %q", got, tt.want)
			}
		})
	}
}
