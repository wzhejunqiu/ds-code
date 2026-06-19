package scroll

import "time"

// Target identifies which viewport owns scroll pending state.
type Target int

const (
	TargetChat Target = iota
	TargetTool
)

// Controller tracks wheel pending scroll and drain activity.
type Controller struct {
	ChatPending int
	ToolPending int
	Active      bool

	profile     Profile
	lastWheelAt time.Time
	wheelFrac   float64
}

// NewController builds a controller using the current terminal profile.
func NewController() Controller {
	return Controller{profile: DetectProfile()}
}

func (c *Controller) pendingPtr(target Target) *int {
	switch target {
	case TargetChat:
		return &c.ChatPending
	case TargetTool:
		return &c.ToolPending
	default:
		return nil
	}
}

// ScrollBy accumulates pending delta without changing viewport offset.
func (c *Controller) ScrollBy(target Target, dy int) {
	if dy == 0 {
		return
	}
	p := c.pendingPtr(target)
	if p == nil {
		return
	}
	*p = ClampPending(*p + dy)
}

// ClearAll resets all pending scroll state.
func (c *Controller) ClearAll() {
	c.ChatPending = 0
	c.ToolPending = 0
	c.Active = false
}

// ClearTarget clears pending for one viewport.
func (c *Controller) ClearTarget(target Target) {
	if p := c.pendingPtr(target); p != nil {
		*p = 0
	}
	if !c.HasPending() {
		c.Active = false
	}
}

// HasPending reports whether any viewport has queued scroll.
func (c *Controller) HasPending() bool {
	return c.ChatPending != 0 || c.ToolPending != 0
}

// ActiveTarget returns the target with non-zero pending (chat first).
func (c *Controller) ActiveTarget() (Target, int, bool) {
	switch {
	case c.ChatPending != 0:
		return TargetChat, c.ChatPending, true
	case c.ToolPending != 0:
		return TargetTool, c.ToolPending, true
	default:
		return TargetChat, 0, false
	}
}

// ApplyDrain subtracts applied scroll from pending.
func (c *Controller) ApplyDrain(target Target, applied int) {
	if applied == 0 {
		return
	}
	p := c.pendingPtr(target)
	if p == nil {
		return
	}
	if *p > 0 {
		*p -= applied
		if *p < 0 {
			*p = 0
		}
	} else if *p < 0 {
		*p += applied
		if *p > 0 {
			*p = 0
		}
	}
	if !c.HasPending() {
		c.Active = false
	}
}

// JumpYOffset computes the clamped offset after a page jump (includes pending).
func JumpYOffset(current, pending, delta, maxY int) int {
	target := current + pending + delta
	if target < 0 {
		return 0
	}
	if target > maxY {
		return maxY
	}
	return target
}
