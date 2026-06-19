package scroll

const (
	ScrollMinPerFrame      = 4
	ScrollInstantThreshold = 5
	ScrollStepMed          = 2
	ScrollStepHigh         = 3
	ScrollMaxPending       = 30
	PendingMax             = 48
)

// DrainStep returns how many lines to scroll this frame for the given pending delta.
func DrainStep(profile Profile, pending, viewportH int) int {
	if pending == 0 || viewportH < 1 {
		return 0
	}
	abs := pending
	if abs < 0 {
		abs = -abs
	}
	var step int
	switch profile {
	case ProfileIntegrated:
		step = drainAdaptive(abs, viewportH)
	default:
		step = drainProportional(abs, viewportH)
	}
	if step > abs {
		step = abs
	}
	if pending < 0 {
		return -step
	}
	return step
}

func drainProportional(absPending, viewportH int) int {
	step := absPending * 3 / 4
	if step < ScrollMinPerFrame {
		step = ScrollMinPerFrame
	}
	cap := viewportH - 1
	if cap < 1 {
		cap = 1
	}
	if step > cap {
		step = cap
	}
	if step > absPending {
		step = absPending
	}
	return step
}

func drainAdaptive(absPending, viewportH int) int {
	if absPending <= ScrollInstantThreshold {
		return absPending
	}
	if absPending > ScrollMaxPending {
		absPending = ScrollMaxPending
	}
	step := ScrollStepMed
	if absPending > 15 {
		step = ScrollStepHigh
	}
	cap := viewportH - 1
	if cap < 1 {
		cap = 1
	}
	if step > cap {
		step = cap
	}
	if step > absPending {
		step = absPending
	}
	return step
}

// SnapPending caps accumulated wheel pending before drain.
func SnapPending(n int) int {
	if n > ScrollMaxPending {
		return ScrollMaxPending
	}
	if n < -ScrollMaxPending {
		return -ScrollMaxPending
	}
	return n
}

// ClampPending caps the running pending queue.
func ClampPending(n int) int {
	if n > PendingMax {
		return PendingMax
	}
	if n < -PendingMax {
		return -PendingMax
	}
	return n
}
