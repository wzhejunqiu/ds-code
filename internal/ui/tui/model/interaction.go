package model

import (
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

// chatInteractionEnabled reports whether chat-area mouse wheel and text selection are allowed.
func (m *Model) chatInteractionEnabled() bool {
	if m.Overlay != state.OverlayNone || m.Prompt != nil {
		return false
	}
	return true
}
