package input

import (
	"regexp"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// leakedSGRMouseRe matches xterm SGR mouse events missing the leading ESC prefix.
// Example: [<64;91;6M (wheel up at x=91, y=6).
var leakedSGRMouseRe = regexp.MustCompile(`\[<(\d+);(\d+);(\d+)([Mm])`)

// leakedSGRPrefixRe matches the partial payload after "[<" while a sequence is still arriving.
var leakedSGRPrefixRe = regexp.MustCompile(`^\d*(;\d*(;\d*)?)?$`)

const (
	sgrBitWheel = 0b0100_0000
	sgrBitsMask = 0b0000_0011
)

// RecoverLeakedMouseKeys detects SGR mouse escape sequences that Bubble Tea
// failed to parse when the entire payload arrives in a single KeyRunes message.
func RecoverLeakedMouseKeys(msg tea.KeyMsg) ([]tea.MouseMsg, bool) {
	var buf string
	events, passthrough, pending := AccumulateLeakedMouseKeys(&buf, msg)
	if pending || len(passthrough.Runes) > 0 || len(events) == 0 {
		return nil, false
	}
	return events, true
}

// AccumulateLeakedMouseKeys reassembles SGR mouse escape sequences split across
// multiple KeyRunes messages (common in iTerm2). When pending is true the chunk
// is fully consumed and must not reach textinput.
func AccumulateLeakedMouseKeys(buf *string, msg tea.KeyMsg) ([]tea.MouseMsg, tea.KeyMsg, bool) {
	if msg.Type != tea.KeyRunes || len(msg.Runes) == 0 {
		return nil, msg, false
	}

	*buf += string(msg.Runes)
	events := extractLeakedSGREvents(buf)

	if isLeakedSGRPrefix(*buf) {
		return events, tea.KeyMsg{}, true
	}

	if *buf == "" {
		return events, tea.KeyMsg{}, false
	}

	passthrough := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(*buf)}
	*buf = ""
	return events, passthrough, false
}

func extractLeakedSGREvents(buf *string) []tea.MouseMsg {
	var out []tea.MouseMsg
	for {
		loc := leakedSGRMouseRe.FindStringIndex(*buf)
		if loc == nil || loc[0] != 0 {
			break
		}
		sub := leakedSGRMouseRe.FindStringSubmatch(*buf)
		btn, _ := strconv.Atoi(sub[1])
		x, _ := strconv.Atoi(sub[2])
		y, _ := strconv.Atoi(sub[3])
		ev := parseLeakedSGRButton(btn)
		ev.X = x - 1
		ev.Y = y - 1
		out = append(out, tea.MouseMsg(ev))
		*buf = (*buf)[loc[1]:]
	}
	return out
}

func isLeakedSGRPrefix(s string) bool {
	switch {
	case s == "":
		return false
	case s == "[":
		return true
	case len(s) < 2 || s[0] != '[' || s[1] != '<':
		return false
	case leakedSGRMouseRe.MatchString(s):
		return false
	default:
		return leakedSGRPrefixRe.MatchString(s[2:])
	}
}

func parseLeakedSGRButton(b int) tea.MouseEvent {
	var m tea.MouseEvent
	m.Action = tea.MouseActionPress

	if b&sgrBitWheel != 0 {
		m.Button = tea.MouseButtonWheelUp + tea.MouseButton(b&sgrBitsMask)
		return m
	}

	m.Button = tea.MouseButtonLeft + tea.MouseButton(b&sgrBitsMask)
	return m
}
