package input

import (
	"regexp"
	"strconv"

	tea "charm.land/bubbletea/v2"
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
// failed to parse when the entire payload arrives in a single KeyPressMsg.
func RecoverLeakedMouseKeys(msg tea.KeyPressMsg) ([]tea.Msg, bool) {
	var buf string
	events, passthrough, pending := AccumulateLeakedMouseKeys(&buf, msg)
	if pending || passthrough.Text != "" || len(events) == 0 {
		return nil, false
	}
	return events, true
}

// AccumulateLeakedMouseKeys reassembles SGR mouse escape sequences split across
// multiple KeyPressMsg messages (common in iTerm2). When pending is true the chunk
// is fully consumed and must not reach textinput.
func AccumulateLeakedMouseKeys(buf *string, msg tea.KeyPressMsg) ([]tea.Msg, tea.KeyPressMsg, bool) {
	if msg.Text == "" {
		return nil, msg, false
	}

	*buf += msg.Text
	events := extractLeakedSGREvents(buf)

	if isLeakedSGRPrefix(*buf) {
		return events, tea.KeyPressMsg{}, true
	}

	if *buf == "" {
		return events, tea.KeyPressMsg{}, false
	}

	passthrough := tea.KeyPressMsg{Text: *buf, Code: tea.KeyExtended}
	*buf = ""
	return events, passthrough, false
}

func extractLeakedSGREvents(buf *string) []tea.Msg {
	var out []tea.Msg
	for {
		loc := leakedSGRMouseRe.FindStringIndex(*buf)
		if loc == nil || loc[0] != 0 {
			break
		}
		sub := leakedSGRMouseRe.FindStringSubmatch(*buf)
		btn, _ := strconv.Atoi(sub[1])
		x, _ := strconv.Atoi(sub[2])
		y, _ := strconv.Atoi(sub[3])
		wheel := parseLeakedSGRWheel(btn)
		wheel.X = x - 1
		wheel.Y = y - 1
		out = append(out, wheel)
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

func parseLeakedSGRWheel(b int) tea.MouseWheelMsg {
	if b&sgrBitWheel != 0 {
		return tea.MouseWheelMsg{Button: tea.MouseWheelUp + tea.MouseButton(b&sgrBitsMask)}
	}
	return tea.MouseWheelMsg{Button: tea.MouseLeft + tea.MouseButton(b&sgrBitsMask)}
}
