package agent

import "testing"

func TestLoopState_initial(t *testing.T) {
	st := &LoopState{}
	if st.Phase != 0 {
		t.Fatalf("expected zero phase, got %v", st.Phase)
	}
	if st.CompactRetried || st.SnipRetried || st.MaxTokensEscalated || st.FallbackTried {
		t.Fatal("expected fresh recovery flags to be false")
	}
}

func TestTransition_constants(t *testing.T) {
	if TransCompactRetry != "compact_retry" {
		t.Fatalf("unexpected transition constant: %q", TransCompactRetry)
	}
}
