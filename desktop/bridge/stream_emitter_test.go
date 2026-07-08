package bridge_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/desktop/bridge"
)

func TestStreamEmitterBatchingAndFlush(t *testing.T) {
	var emitted []bridge.AgentEventEnvelope
	sink := make(chan struct{}, 64)
	emit := func(env bridge.AgentEventEnvelope) bool {
		emitted = append(emitted, env)
		select {
		case sink <- struct{}{}:
		default:
		}
		return true
	}
	em := bridge.NewStreamEmitter(bridge.StreamEmitterOptions{
		TurnID:     "turn-1",
		Emit:       emit,
		FlushEvery: 16 * time.Millisecond,
		MaxChunk:   8192,
	})

	em.OnDelta(bridge.KindContentDelta, "hello ")
	em.OnDelta(bridge.KindContentDelta, "world")
	em.Flush(true)

	if len(emitted) == 0 {
		t.Fatal("expected at least one emit after flush")
	}
	found := false
	for _, env := range emitted {
		if env.Kind != bridge.KindContentDelta {
			continue
		}
		var p bridge.ContentDeltaPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			t.Fatal(err)
		}
		if p.Delta == "hello world" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected batched delta, got %d events", len(emitted))
	}
}

func TestStreamEmitterCriticalNotDropped(t *testing.T) {
	block := make(chan struct{})
	var count int
	emit := func(env bridge.AgentEventEnvelope) bool {
		count++
		if env.Critical {
			<-block
		}
		return !env.Critical
	}
	em := bridge.NewStreamEmitter(bridge.StreamEmitterOptions{TurnID: "t", Emit: emit})
	go func() {
		time.Sleep(20 * time.Millisecond)
		close(block)
	}()
	em.EmitCritical(bridge.KindToolEnd, bridge.ToolEndPayload{Name: "read", Result: "ok"})
	if count == 0 {
		t.Fatal("expected critical emit attempt")
	}
}

func TestGoldenTurnSequence(t *testing.T) {
	var emitted []bridge.AgentEventEnvelope
	emit := func(env bridge.AgentEventEnvelope) bool {
		emitted = append(emitted, env)
		return true
	}
	em := bridge.NewStreamEmitter(bridge.StreamEmitterOptions{TurnID: "golden-turn", Emit: emit})
	bridge.EmitTurnStarted(em, "sess-1", "markdown")
	cb := bridge.TurnCallbacks(bridge.TurnCallbacksOptions{Emitter: em, SessionID: "sess-1"})
	cb.OnContentDelta("Hi")
	cb.OnAssistantSegmentEnd()
	cb.OnToolStart("read", "{}", "", time.Time{})
	cb.OnToolEnd("read", "{}", "", "content", false)
	bridge.EmitTurnDone(em, nil, nil)

	goldenPath := filepath.Join("testdata", "golden_turn.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		b, _ := json.MarshalIndent(emitted, "", "  ")
		_ = os.MkdirAll("testdata", 0o755)
		_ = os.WriteFile(goldenPath, b, 0o644)
	}
	raw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (run with UPDATE_GOLDEN=1)", err)
	}
	var want []bridge.AgentEventEnvelope
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) != len(emitted) {
		t.Fatalf("event count: got %d want %d", len(emitted), len(want))
	}
	for i := range want {
		if want[i].Kind != emitted[i].Kind {
			t.Fatalf("event %d kind: got %q want %q", i, emitted[i].Kind, want[i].Kind)
		}
		if want[i].Critical != emitted[i].Critical {
			t.Fatalf("event %d critical mismatch", i)
		}
	}
}
