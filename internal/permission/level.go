package permission

// Level classifies tool risk.
type Level int

const (
	LevelLow Level = iota
	LevelMedium
	LevelHigh
	LevelHighest
)
