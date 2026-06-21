package turn

import (
	"time"

	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

// NeedsBashTimeoutTick reports whether a running sync bash tool needs live countdown refresh.
func NeedsBashTimeoutTick(s *state.State) bool {
	if !s.Running {
		return false
	}
	now := time.Now()
	for i := range s.Chat {
		b := &s.Chat[i]
		if b.Role != chat.RoleTool || !b.ToolRunning || !tool.IsShellDisplay(b.ToolName) {
			continue
		}
		if !b.ToolTimeoutDeadline.IsZero() && now.Before(b.ToolTimeoutDeadline) {
			return true
		}
	}
	return false
}
