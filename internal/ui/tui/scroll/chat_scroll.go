package scroll

// ChatBottomSentinel marks sticky-to-bottom scroll mode in chatScrollY.
const ChatBottomSentinel = 1 << 30

// IsPinnedBottom reports whether y is the sticky-to-bottom sentinel.
func IsPinnedBottom(y int) bool {
	return y >= ChatBottomSentinel
}

// EffectiveChatY resolves the sentinel to maxY and clamps y into [0, maxY].
func EffectiveChatY(y, maxY int) int {
	if IsPinnedBottom(y) {
		return maxY
	}
	if y < 0 {
		return 0
	}
	if y > maxY {
		return maxY
	}
	return y
}
