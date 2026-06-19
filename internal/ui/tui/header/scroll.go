package header

// NeedsAutoScroll reports whether the notification zone has more lines than fit on screen.
func NeedsAutoScroll(notices []Notice, termWidth int, narrow bool) bool {
	return MaxScrollOffset(notices, ZoneWidth(termWidth, narrow)) > 0
}

// AdvanceScrollOffset moves to the next notice window, wrapping to 0 at the end.
// Returns false when scrolling is not needed.
func AdvanceScrollOffset(notices []Notice, termWidth int, narrow bool, offset *int) bool {
	if offset == nil || len(notices) == 0 {
		return false
	}
	zoneW := ZoneWidth(termWidth, narrow)
	maxOff := MaxScrollOffset(notices, zoneW)
	if maxOff <= 0 {
		*offset = 0
		return false
	}
	*offset++
	if *offset > maxOff {
		*offset = 0
	}
	return true
}
