package chat

import (
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestBlockNeedsLiveNow_bashCountdown(t *testing.T) {
	now := time.Now()
	b := &Block{
		Role:                RoleTool,
		ToolName:            tool.NameShell.String(),
		ToolRunning:         true,
		ToolTimeoutDeadline: now.Add(time.Minute),
	}
	if !blockNeedsLiveNow(b, now) {
		t.Fatal("expected live refresh for running bash countdown")
	}
}
