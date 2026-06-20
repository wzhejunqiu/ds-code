package component

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestPickerViewHighlightsSelection(t *testing.T) {
	p := Picker{
		Items: []string{"/clear — new session", "/context — context panel"},
	}
	p.Cursor = 1

	lines := strings.Split(p.View(), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), p.View())
	}
	if lines[0] == lines[1] {
		t.Fatal("expected selected and unselected lines to differ visually")
	}
	if !strings.Contains(lines[1], "context") {
		t.Fatalf("selected line missing item: %q", lines[1])
	}
}

func TestPickerEnsureScrollVisible(t *testing.T) {
	p := Picker{PageSize: 5}
	p.SetItems(make([]string, 25))
	for i := range p.Items {
		p.Items[i] = "row"
	}

	p.Cursor = 0
	p.Scroll = 0
	p.ensureScrollVisible()
	if p.Scroll != 0 {
		t.Fatalf("scroll = %d, want 0", p.Scroll)
	}

	p.Cursor = 20
	p.Scroll = 0
	p.ensureScrollVisible()
	if p.Cursor < p.Scroll || p.Cursor >= p.Scroll+p.PageSize {
		t.Fatalf("idx=%d scroll=%d page=%d, selection not in window", p.Cursor, p.Scroll, p.PageSize)
	}
	if p.Scroll == 0 {
		t.Fatal("expected scroll offset to advance for selection near end")
	}

	p.Cursor = 24
	p.ensureScrollVisible()
	maxScroll := 25 - p.PageSize
	if p.Scroll != maxScroll {
		t.Fatalf("scroll = %d, want %d", p.Scroll, maxScroll)
	}
}

func TestPickerMovePageScrollsWindow(t *testing.T) {
	p := Picker{PageSize: 5}
	p.SetItems(make([]string, 20))
	for i := range p.Items {
		p.Items[i] = "row"
	}

	steps := p.PageSize + 2
	for i := 0; i < steps; i++ {
		p.Move(1)
	}
	if p.Scroll == 0 {
		t.Fatal("expected window to scroll down after many down presses")
	}
	if p.Cursor != steps {
		t.Fatalf("cursor = %d, want %d", p.Cursor, steps)
	}
}

func TestPickerHandleKeyTabSelectsFirst(t *testing.T) {
	p := Picker{Items: []string{"a", "b"}}
	p.Cursor = 1

	action, handled := p.HandleKey(tea.KeyPressMsg{Code: tea.KeyTab}, PickerKeyOpts{Tab: PickerTabSelectFirst})
	if !handled || action != PickerKeyConfirmFirst {
		t.Fatalf("action=%v handled=%v", action, handled)
	}
}

func TestPickerHandleKeyTabConfirmsCursor(t *testing.T) {
	p := Picker{Items: []string{"a", "b"}}
	p.Cursor = 1

	action, handled := p.HandleKey(tea.KeyPressMsg{Code: tea.KeyTab}, PickerKeyOpts{Tab: PickerTabConfirm})
	if !handled || action != PickerKeyConfirm {
		t.Fatalf("action=%v handled=%v, want PickerKeyConfirm", action, handled)
	}
}

func TestPickerHandleKeyTabMovesDown(t *testing.T) {
	p := Picker{Items: []string{"a", "b"}}
	p.Cursor = 0

	_, handled := p.HandleKey(tea.KeyPressMsg{Code: tea.KeyTab}, PickerKeyOpts{Tab: PickerTabMoveDown})
	if !handled {
		t.Fatal("expected tab to be handled")
	}
	if p.Cursor != 1 {
		t.Fatalf("cursor = %d, want 1", p.Cursor)
	}
}

func TestPickerHandleKeyEscCancel(t *testing.T) {
	p := Picker{Items: []string{"a"}}
	action, handled := p.HandleKey(tea.KeyPressMsg{Code: tea.KeyEsc}, PickerKeyOpts{})
	if !handled || action != PickerKeyCancel {
		t.Fatalf("action=%v handled=%v", action, handled)
	}
}

func TestPickerHandleKeyEnterConfirm(t *testing.T) {
	p := Picker{Items: []string{"a", "b"}}
	p.Cursor = 1
	action, handled := p.HandleKey(tea.KeyPressMsg{Code: tea.KeyEnter}, PickerKeyOpts{})
	if !handled || action != PickerKeyConfirm {
		t.Fatalf("action=%v handled=%v", action, handled)
	}
}

func TestPickerHandleKeyEnterEmptyList(t *testing.T) {
	p := Picker{}
	action, handled := p.HandleKey(tea.KeyPressMsg{Code: tea.KeyEnter}, PickerKeyOpts{})
	if !handled || action != PickerKeyNone {
		t.Fatalf("action=%v handled=%v", action, handled)
	}
}

func TestPickerHandleKeyPageKeys(t *testing.T) {
	p := Picker{PageSize: 3}
	p.SetItems([]string{"1", "2", "3", "4", "5", "6", "7"})

	_, handled := p.HandleKey(tea.KeyPressMsg{Code: tea.KeyPgDown}, PickerKeyOpts{})
	if !handled {
		t.Fatal("expected pgdown to be handled")
	}
	if p.Cursor != 3 {
		t.Fatalf("cursor = %d, want 3 after pgdown", p.Cursor)
	}

	_, handled = p.HandleKey(tea.KeyPressMsg{Code: tea.KeyPgUp}, PickerKeyOpts{})
	if !handled {
		t.Fatal("expected pgup to be handled")
	}
	if p.Cursor != 0 {
		t.Fatalf("cursor = %d, want 0 after pgup", p.Cursor)
	}
}

func TestPickerHandleKeyPageKeysIgnoredWithoutPageSize(t *testing.T) {
	p := Picker{Items: []string{"a", "b", "c"}}
	_, handled := p.HandleKey(tea.KeyPressMsg{Code: tea.KeyPgDown}, PickerKeyOpts{})
	if handled {
		t.Fatal("expected pgdown to be ignored when PageSize is 0")
	}
}

func TestPickerClearResetsPageSize(t *testing.T) {
	p := Picker{PageSize: 8, Items: []string{"a"}}
	p.Clear()
	if p.PageSize != 0 {
		t.Fatalf("PageSize = %d, want 0", p.PageSize)
	}
}
