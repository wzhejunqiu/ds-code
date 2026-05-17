package agent

import (
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/llm"
)

func TestStreamTiming_duration(t *testing.T) {
	var st streamTiming
	st.reasoningStart = time.Now().Add(-1500 * time.Millisecond)
	st.reasoningEnd = time.Now().Add(-200 * time.Millisecond)
	d := st.duration()
	if d < 1200*time.Millisecond || d > 1800*time.Millisecond {
		t.Fatalf("duration = %v, want ~1.3s", d)
	}
}

func TestStreamTiming_observe(t *testing.T) {
	var st streamTiming
	st.observe(llm.StreamDelta{Reasoning: "a"})
	if st.reasoningStart.IsZero() {
		t.Fatal("expected reasoning start")
	}
	st.observe(llm.StreamDelta{Content: "b"})
	if st.reasoningEnd.IsZero() {
		t.Fatal("expected reasoning end")
	}
}
