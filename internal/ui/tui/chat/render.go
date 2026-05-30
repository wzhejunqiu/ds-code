package chat

import (
	"time"
)

// Render formats chat blocks for the main transcript viewport.
func Render(blocks []Block, width int, now time.Time, showToolDetails bool) string {
	return RenderCached(blocks, width, now, showToolDetails, nil, nil)
}
