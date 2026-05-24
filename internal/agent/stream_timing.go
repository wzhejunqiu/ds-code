package agent

import (
	"time"

	"github.com/wzhejunqiu/ds-code/internal/llm"
)

// streamTiming tracks thinking-phase wall time from streaming deltas.
// Reasoning starts at the first reasoning chunk; ends at the first content chunk
// (or at duration() if the model never streams content).
type streamTiming struct {
	reasoningStart time.Time
	reasoningEnd   time.Time
}

func (t *streamTiming) observe(d llm.StreamDelta) {
	if d.Reasoning != "" && t.reasoningStart.IsZero() {
		t.reasoningStart = time.Now()
	}
	if d.Content != "" && !t.reasoningStart.IsZero() && t.reasoningEnd.IsZero() {
		t.reasoningEnd = time.Now()
	}
}

func (t *streamTiming) duration() time.Duration {
	if t.reasoningStart.IsZero() {
		return 0
	}
	end := t.reasoningEnd
	if end.IsZero() {
		end = time.Now()
	}
	d := end.Sub(t.reasoningStart)
	if d < 0 {
		return 0
	}
	return d
}

func durationMS(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	return int64(d / time.Millisecond)
}
