package input

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRecoverLeakedMouseKeys_wheelUp(t *testing.T) {
	msgs, ok := RecoverLeakedMouseKeys(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("[<64;91;6M"),
	})
	if !ok {
		t.Fatal("expected recovery")
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d msgs, want 1", len(msgs))
	}
	ev := tea.MouseEvent(msgs[0])
	if ev.Button != tea.MouseButtonWheelUp {
		t.Fatalf("button = %v, want wheel up", ev.Button)
	}
	if ev.X != 90 || ev.Y != 5 {
		t.Fatalf("pos = (%d,%d), want (90,5)", ev.X, ev.Y)
	}
	if ev.Action != tea.MouseActionPress {
		t.Fatalf("action = %v, want press", ev.Action)
	}
}

func TestRecoverLeakedMouseKeys_wheelDown(t *testing.T) {
	msgs, ok := RecoverLeakedMouseKeys(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("[<65;87;31M"),
	})
	if !ok {
		t.Fatal("expected recovery")
	}
	ev := tea.MouseEvent(msgs[0])
	if ev.Button != tea.MouseButtonWheelDown {
		t.Fatalf("button = %v, want wheel down", ev.Button)
	}
	if ev.X != 86 || ev.Y != 30 {
		t.Fatalf("pos = (%d,%d), want (86,30)", ev.X, ev.Y)
	}
}

func TestRecoverLeakedMouseKeys_concatenated(t *testing.T) {
	msgs, ok := RecoverLeakedMouseKeys(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("[<64;91;6M[<65;87;31M"),
	})
	if !ok {
		t.Fatal("expected recovery")
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d msgs, want 2", len(msgs))
	}
	if tea.MouseEvent(msgs[0]).Button != tea.MouseButtonWheelUp {
		t.Fatalf("first button = %v, want wheel up", tea.MouseEvent(msgs[0]).Button)
	}
	if tea.MouseEvent(msgs[1]).Button != tea.MouseButtonWheelDown {
		t.Fatalf("second button = %v, want wheel down", tea.MouseEvent(msgs[1]).Button)
	}
}

func TestRecoverLeakedMouseKeys_ignoresNormalInput(t *testing.T) {
	tests := []string{"hello", "/context", "foo [< bar", "[not;mouse;seq]"}
	for _, s := range tests {
		if _, ok := RecoverLeakedMouseKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}); ok {
			t.Fatalf("RecoverLeakedMouseKeys(%q) = true, want false", s)
		}
	}
}

func TestRecoverLeakedMouseKeys_ignoresNonRunes(t *testing.T) {
	if _, ok := RecoverLeakedMouseKeys(tea.KeyMsg{Type: tea.KeyEnter}); ok {
		t.Fatal("expected false for KeyEnter")
	}
}

func TestRecoverLeakedMouseKeys_mixedPayloadRejected(t *testing.T) {
	if _, ok := RecoverLeakedMouseKeys(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("x[<64;91;6M"),
	}); ok {
		t.Fatal("expected mixed payload to be rejected")
	}
}

func TestAccumulateLeakedMouseKeys_charByChar(t *testing.T) {
	seq := "[<64;48;25M"
	var buf string
	var all []tea.MouseMsg
	for i, r := range seq {
		events, _, pending := AccumulateLeakedMouseKeys(&buf, tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{r},
		})
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
	ev := tea.MouseEvent(all[0])
	if ev.Button != tea.MouseButtonWheelUp {
		t.Fatalf("button = %v, want wheel up", ev.Button)
	}
	if ev.X != 47 || ev.Y != 24 {
		t.Fatalf("pos = (%d,%d), want (47,24)", ev.X, ev.Y)
	}
}

func TestAccumulateLeakedMouseKeys_twoEventsCharByChar(t *testing.T) {
	seq := "[<64;48;25M[<64;48;25M"
	var buf string
	var all []tea.MouseMsg
	for _, r := range seq {
		events, _, _ := AccumulateLeakedMouseKeys(&buf, tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{r},
		})
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
	_, _, pending := AccumulateLeakedMouseKeys(&buf, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	if !pending {
		t.Fatal("expected pending after [")
	}
	events, passthrough, pending := AccumulateLeakedMouseKeys(&buf, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("foo")})
	if pending {
		t.Fatal("expected not pending after [foo")
	}
	if len(events) != 0 {
		t.Fatalf("events = %d, want 0", len(events))
	}
	if string(passthrough.Runes) != "[foo" {
		t.Fatalf("passthrough = %q, want [foo", string(passthrough.Runes))
	}
	if buf != "" {
		t.Fatalf("buffer = %q, want empty", buf)
	}
}

func TestAccumulateLeakedMouseKeys_ignoresNonRunes(t *testing.T) {
	var buf string
	_, passthrough, pending := AccumulateLeakedMouseKeys(&buf, tea.KeyMsg{Type: tea.KeyEnter})
	if pending || buf != "" {
		t.Fatal("expected no accumulation for KeyEnter")
	}
	if passthrough.Type != tea.KeyEnter {
		t.Fatalf("passthrough type = %v, want KeyEnter", passthrough.Type)
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
