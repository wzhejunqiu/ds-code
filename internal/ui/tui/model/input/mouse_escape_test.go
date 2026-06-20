package input

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func keyText(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: s, Code: tea.KeyExtended}
}

func TestRecoverLeakedMouseKeys_wheelUp(t *testing.T) {
	msgs, ok := RecoverLeakedMouseKeys(keyText("[<64;91;6M"))
	if !ok {
		t.Fatal("expected recovery")
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d msgs, want 1", len(msgs))
	}
	wheel, ok := msgs[0].(tea.MouseWheelMsg)
	if !ok {
		t.Fatalf("got %T, want MouseWheelMsg", msgs[0])
	}
	if wheel.Button != tea.MouseWheelUp {
		t.Fatalf("button = %v, want wheel up", wheel.Button)
	}
	if wheel.X != 90 || wheel.Y != 5 {
		t.Fatalf("pos = (%d,%d), want (90,5)", wheel.X, wheel.Y)
	}
}

func TestRecoverLeakedMouseKeys_wheelDown(t *testing.T) {
	msgs, ok := RecoverLeakedMouseKeys(keyText("[<65;87;31M"))
	if !ok {
		t.Fatal("expected recovery")
	}
	wheel, ok := msgs[0].(tea.MouseWheelMsg)
	if !ok {
		t.Fatalf("got %T, want MouseWheelMsg", msgs[0])
	}
	if wheel.Button != tea.MouseWheelDown {
		t.Fatalf("button = %v, want wheel down", wheel.Button)
	}
	if wheel.X != 86 || wheel.Y != 30 {
		t.Fatalf("pos = (%d,%d), want (86,30)", wheel.X, wheel.Y)
	}
}

func TestRecoverLeakedMouseKeys_concatenated(t *testing.T) {
	msgs, ok := RecoverLeakedMouseKeys(keyText("[<64;91;6M[<65;87;31M"))
	if !ok {
		t.Fatal("expected recovery")
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d msgs, want 2", len(msgs))
	}
	if w, ok := msgs[0].(tea.MouseWheelMsg); !ok || w.Button != tea.MouseWheelUp {
		t.Fatalf("first button = %v, want wheel up", msgs[0])
	}
	if w, ok := msgs[1].(tea.MouseWheelMsg); !ok || w.Button != tea.MouseWheelDown {
		t.Fatalf("second button = %v, want wheel down", msgs[1])
	}
}

func TestRecoverLeakedMouseKeys_ignoresNormalInput(t *testing.T) {
	tests := []string{"hello", "/context", "foo [< bar", "[not;mouse;seq]"}
	for _, s := range tests {
		if _, ok := RecoverLeakedMouseKeys(keyText(s)); ok {
			t.Fatalf("RecoverLeakedMouseKeys(%q) = true, want false", s)
		}
	}
}

func TestRecoverLeakedMouseKeys_ignoresNonRunes(t *testing.T) {
	if _, ok := RecoverLeakedMouseKeys(tea.KeyPressMsg{Code: tea.KeyEnter}); ok {
		t.Fatal("expected false for KeyEnter")
	}
}

func TestRecoverLeakedMouseKeys_mixedPayloadRejected(t *testing.T) {
	if _, ok := RecoverLeakedMouseKeys(keyText("x[<64;91;6M")); ok {
		t.Fatal("expected mixed payload to be rejected")
	}
}

func TestAccumulateLeakedMouseKeys_charByChar(t *testing.T) {
	seq := "[<64;48;25M"
	var buf string
	var all []tea.Msg
	for i, r := range seq {
		events, _, pending := AccumulateLeakedMouseKeys(&buf, keyText(string(r)))
		all = append(all, events...)
		isLast := i == len(seq)-1
		if isLast {
			if pending {
				t.Fatal("expected not pending on final char")
			}
		} else if !pending {
			t.Fatalf("expected pending before final char, got flush at %q with buf %q", string(r), buf)
		}
	}
	if buf != "" {
		t.Fatalf("buffer = %q, want empty", buf)
	}
	if len(all) != 1 {
		t.Fatalf("got %d events, want 1", len(all))
	}
	wheel, ok := all[0].(tea.MouseWheelMsg)
	if !ok || wheel.Button != tea.MouseWheelUp {
		t.Fatalf("button = %v, want wheel up", all[0])
	}
	if wheel.X != 47 || wheel.Y != 24 {
		t.Fatalf("pos = (%d,%d), want (47,24)", wheel.X, wheel.Y)
	}
}

func TestAccumulateLeakedMouseKeys_twoEventsCharByChar(t *testing.T) {
	seq := "[<64;48;25M[<64;48;25M"
	var buf string
	var all []tea.Msg
	for _, r := range seq {
		events, _, _ := AccumulateLeakedMouseKeys(&buf, keyText(string(r)))
		all = append(all, events...)
	}
	if buf != "" {
		t.Fatalf("buffer = %q, want empty", buf)
	}
	if len(all) != 2 {
		t.Fatalf("got %d events, want 2", len(all))
	}
}

func TestAccumulateLeakedMouseKeys_bracketThenNormalText(t *testing.T) {
	var buf string
	_, _, pending := AccumulateLeakedMouseKeys(&buf, keyText("["))
	if !pending {
		t.Fatal("expected pending after [")
	}
	events, passthrough, pending := AccumulateLeakedMouseKeys(&buf, keyText("foo"))
	if pending {
		t.Fatal("expected not pending after [foo")
	}
	if len(events) != 0 {
		t.Fatalf("events = %d, want 0", len(events))
	}
	if passthrough.Text != "[foo" {
		t.Fatalf("passthrough = %q, want [foo", passthrough.Text)
	}
	if buf != "" {
		t.Fatalf("buffer = %q, want empty", buf)
	}
}

func TestAccumulateLeakedMouseKeys_ignoresNonRunes(t *testing.T) {
	var buf string
	_, passthrough, pending := AccumulateLeakedMouseKeys(&buf, tea.KeyPressMsg{Code: tea.KeyEnter})
	if pending || buf != "" {
		t.Fatal("expected no accumulation for KeyEnter")
	}
	if passthrough.Code != tea.KeyEnter {
		t.Fatalf("passthrough code = %v, want KeyEnter", passthrough.Code)
	}
}

func TestIsLeakedSGRPrefix(t *testing.T) {
	prefixes := []string{"[", "[<", "[<64", "[<64;", "[<64;48", "[<64;48;", "[<64;48;25"}
	for _, s := range prefixes {
		if !isLeakedSGRPrefix(s) {
			t.Fatalf("isLeakedSGRPrefix(%q) = false, want true", s)
		}
	}
	nonPrefixes := []string{"", "hello", "[foo", "[not;mouse;seq]"}
	for _, s := range nonPrefixes {
		if isLeakedSGRPrefix(s) {
			t.Fatalf("isLeakedSGRPrefix(%q) = true, want false", s)
		}
	}
}
