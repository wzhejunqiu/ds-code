package chat

import (
	"time"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

// Render formats chat blocks for the main transcript viewport.
func Render(blocks []Block, width int, now time.Time, showToolDetails bool, disp tool.DisplayContext) string {
	return RenderCached(blocks, width, now, showToolDetails, disp, nil, nil)
}
