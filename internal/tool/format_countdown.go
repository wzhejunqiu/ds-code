package tool

import (
	"fmt"
	"time"
)

// FormatTimeoutCountdown formats remaining time until deadline for TUI display.
func FormatTimeoutCountdown(deadline, now time.Time) string {
	if deadline.IsZero() {
		return ""
	}
	rem := deadline.Sub(now)
	if rem <= 0 {
		return "0s"
	}
	secs := int(rem.Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	mins := secs / 60
	s := secs % 60
	return fmt.Sprintf("%d:%02d", mins, s)
}
