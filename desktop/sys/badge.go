package sys

import (
	"sync/atomic"
)

var (
	runningTurns atomic.Int32
	pendingPerms atomic.Int32
)

// IncRunningTurn increments active turn count.
func IncRunningTurn() {
	runningTurns.Add(1)
}

// DecRunningTurn decrements active turn count.
func DecRunningTurn() {
	runningTurns.Add(-1)
}

// SetWaitingPermission updates waiting-permission count for badge.
func SetWaitingPermission(waiting bool) {
	if waiting {
		pendingPerms.Store(1)
	} else {
		pendingPerms.Store(0)
	}
}

// BadgeCount returns dock badge count (running + waiting permission).
func BadgeCount() int {
	n := int(runningTurns.Load())
	if pendingPerms.Load() > 0 {
		n++
	}
	return n
}
