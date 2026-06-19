package scroll

import (
	"os"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	wheelNotchLines  = 3
	trackpadBurstGap = 40 * time.Millisecond
)

// WheelSpeed returns the scroll speed multiplier from DS_CODE_SCROLL_SPEED (default 1).
func WheelSpeed() float64 {
	raw := os.Getenv("DS_CODE_SCROLL_SPEED")
	if raw == "" {
		return 1
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return 1
	}
	return v
}

// ComputeWheelStep returns signed lines to accumulate for a wheel event.
func (c *Controller) ComputeWheelStep(msg tea.MouseMsg, now time.Time) int {
	dir := wheelDirection(msg)
	if dir == 0 {
		return 0
	}
	speed := WheelSpeed()
	var step float64
	switch c.profile {
	case ProfileIntegrated:
		step = integratedWheelStep(c, dir, speed)
	default:
		step = nativeWheelStep(c, dir, speed, now)
	}
	if step == 0 {
		if dir > 0 {
			return 1
		}
		return -1
	}
	n := int(step)
	if n == 0 {
		if dir > 0 {
			return 1
		}
		return -1
	}
	return n
}

func wheelDirection(msg tea.MouseMsg) int {
	switch msg.Button {
	case tea.MouseButtonWheelDown:
		return 1
	case tea.MouseButtonWheelUp:
		return -1
	default:
		return 0
	}
}

func nativeWheelStep(c *Controller, dir int, speed float64, now time.Time) float64 {
	base := float64(wheelNotchLines) * speed
	if !c.lastWheelAt.IsZero() && now.Sub(c.lastWheelAt) <= trackpadBurstGap {
		base *= 1.5
	}
	c.lastWheelAt = now
	if dir < 0 {
		return -base
	}
	return base
}

func integratedWheelStep(c *Controller, dir int, speed float64) float64 {
	base := float64(wheelNotchLines) * speed * 0.85
	c.wheelFrac += base
	n := int(c.wheelFrac)
	if n == 0 {
		n = 1
		c.wheelFrac = 0
	} else {
		c.wheelFrac -= float64(n)
	}
	if dir < 0 {
		return -float64(n)
	}
	return float64(n)
}
