package component

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Picker is a keyboard-navigable overlay list (slash completion, /resume, etc.).
type Picker struct {
	Header   string
	Empty    string
	Items    []string
	Cursor   int
	Scroll   int
	PageSize int // 0 shows all items
}

// PickerTabBehavior configures Tab key handling for a picker instance.
type PickerTabBehavior int

const (
	// PickerTabDefault leaves Tab to the text input (not handled by the picker).
	PickerTabDefault PickerTabBehavior = iota
	// PickerTabSelectFirst confirms the first item (slash completion).
	PickerTabSelectFirst
	// PickerTabMoveDown moves the cursor down (resume session list).
	PickerTabMoveDown
)

// PickerKeyOpts configures key handling differences between pickers.
type PickerKeyOpts struct {
	Tab PickerTabBehavior
}

// PickerKeyAction is returned when a key should close or confirm the picker.
type PickerKeyAction int

const (
	PickerKeyNone PickerKeyAction = iota
	PickerKeyCancel
	PickerKeyConfirm
	PickerKeyConfirmFirst
)

func (p *Picker) Len() int {
	return len(p.Items)
}

func (p *Picker) Clear() {
	p.Header = ""
	p.Empty = ""
	p.Items = nil
	p.Cursor = 0
	p.Scroll = 0
	p.PageSize = 0
}

func (p *Picker) SetItems(items []string) {
	p.Items = items
}

func (p *Picker) ResetSelection() {
	p.Cursor = 0
	p.Scroll = 0
}

func (p *Picker) ClampSelection() {
	if p.Len() == 0 {
		p.Cursor = 0
		p.Scroll = 0
		return
	}
	if p.Cursor < 0 {
		p.Cursor = 0
	}
	if p.Cursor >= p.Len() {
		p.Cursor = p.Len() - 1
	}
	p.ensureScrollVisible()
}

func (p *Picker) Move(delta int) {
	if p.Len() == 0 {
		return
	}
	p.Cursor += delta
	p.ensureScrollVisible()
}

func (p *Picker) MovePage(pages int) {
	if p.Len() == 0 || p.PageSize <= 0 {
		return
	}
	p.Cursor += pages * p.PageSize
	p.ensureScrollVisible()
}

func (p *Picker) HandleKey(msg tea.KeyMsg, opts PickerKeyOpts) (PickerKeyAction, bool) {
	switch msg.String() {
	case "up":
		p.Move(-1)
		return PickerKeyNone, true
	case "down":
		p.Move(1)
		return PickerKeyNone, true
	case "tab":
		switch opts.Tab {
		case PickerTabSelectFirst:
			if p.Len() > 0 {
				return PickerKeyConfirmFirst, true
			}
			return PickerKeyNone, true
		case PickerTabMoveDown:
			p.Move(1)
			return PickerKeyNone, true
		default:
			return PickerKeyNone, false
		}
	case "pgup":
		if p.PageSize > 0 {
			p.MovePage(-1)
			return PickerKeyNone, true
		}
	case "pgdown":
		if p.PageSize > 0 {
			p.MovePage(1)
			return PickerKeyNone, true
		}
	case "enter":
		if p.Len() > 0 {
			return PickerKeyConfirm, true
		}
		return PickerKeyNone, true
	case "esc":
		return PickerKeyCancel, true
	}
	return PickerKeyNone, false
}

func (p *Picker) View() string {
	if p.Len() == 0 {
		return p.Empty
	}
	p.ensureScrollVisible()

	var b strings.Builder
	if p.Header != "" {
		b.WriteString(p.Header)
		b.WriteByte('\n')
	}

	start, end := p.visibleRange()
	for i := start; i < end; i++ {
		line := listPrefix(i == p.Cursor) + p.Items[i]
		b.WriteString(renderListLine(line, i == p.Cursor))
		b.WriteByte('\n')
	}
	if p.PageSize > 0 && p.Len() > p.PageSize {
		fmt.Fprintf(&b, "  — %d–%d of %d —", start+1, end, p.Len())
	}
	return strings.TrimRight(b.String(), "\n")
}

func (p *Picker) visibleRange() (start, end int) {
	if p.PageSize <= 0 {
		return 0, p.Len()
	}
	start = p.Scroll
	end = p.Scroll + p.PageSize
	if end > p.Len() {
		end = p.Len()
	}
	return start, end
}

func (p *Picker) ensureScrollVisible() {
	total := p.Len()
	if total == 0 {
		p.Scroll = 0
		return
	}
	page := p.pageSize()
	if p.Cursor < 0 {
		p.Cursor = 0
	}
	if p.Cursor >= total {
		p.Cursor = total - 1
	}
	if p.Cursor < p.Scroll {
		p.Scroll = p.Cursor
	}
	if p.Cursor >= p.Scroll+page {
		p.Scroll = p.Cursor - page + 1
	}
	maxScroll := total - page
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.Scroll > maxScroll {
		p.Scroll = maxScroll
	}
	if p.Scroll < 0 {
		p.Scroll = 0
	}
}

func (p *Picker) pageSize() int {
	if p.PageSize <= 0 {
		return p.Len()
	}
	return p.PageSize
}

func listPrefix(selected bool) string {
	if selected {
		return "▸ "
	}
	return "  "
}

func renderListLine(line string, selected bool) string {
	if selected {
		return styleItemSelected.Render(line)
	}
	return styleItem.Render(line)
}
