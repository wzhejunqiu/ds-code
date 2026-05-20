package chat

import (
	"fmt"
	"time"
)

// ThinkingFineDuration is the threshold for sub-second thinking label updates.
const ThinkingFineDuration = 10 * time.Second

func planningBlockLabel(started, now time.Time) string {
	if started.IsZero() {
		return planningBullet + planningLabel
	}
	d := now.Sub(started)
	if d < 0 {
		d = 0
	}
	if d > 0 {
		return planningBullet + planningLabel + "  " + formatThinkingDuration(d)
	}
	return planningBullet + planningLabel
}

// reasoningExpanded reports whether the thinking trace body should be visible.
// Active thinking is always expanded; completed thinking follows ReasoningOpen (e.g. Ctrl+R).
func reasoningExpanded(b Block) bool {
	if b.Role != RoleAssistant {
		return false
	}
	activelyThinking := !b.ReasoningStartedAt.IsZero() && b.ReasoningEndedAt.IsZero()
	return b.ReasoningOpen || activelyThinking
}

func reasoningBlockLabel(open bool, started, ended, now time.Time, fixed time.Duration) string {
	arrow := "▸"
	if open {
		arrow = "▾"
	}
	var d time.Duration
	if fixed > 0 {
		d = fixed
	} else if !started.IsZero() {
		endAt := ended
		if endAt.IsZero() {
			endAt = now
		}
		d = endAt.Sub(started)
		if d < 0 {
			d = 0
		}
	}
	thinkingDone := fixed > 0 || !ended.IsZero()
	if thinkingDone {
		if d > 0 {
			return arrow + " thought for " + formatThinkingDuration(d)
		}
		return arrow + " thought"
	}
	if d > 0 {
		return arrow + " thinking " + formatThinkingDuration(d)
	}
	return arrow + " thinking"
}

func turnDurationLine(d time.Duration) string {
	return "task took " + formatThinkingDuration(d)
}

// FormatThinkingDuration formats a duration for thinking/planning labels.
func FormatThinkingDuration(d time.Duration) string {
	return formatThinkingDuration(d)
}

func formatThinkingDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < ThinkingFineDuration {
		tenths := int(d.Round(100*time.Millisecond) / (100 * time.Millisecond))
		whole, frac := tenths/10, tenths%10
		if frac == 0 {
			return fmt.Sprintf("%ds", whole)
		}
		return fmt.Sprintf("%d.%ds", whole, frac)
	}
	s := int(d.Round(time.Second).Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	s %= 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}
