package session

import "time"

// Summary is a row for session listing.
type Summary struct {
	ID           string
	Title        string
	Model        string
	BilledTokens int
	UpdatedAt    time.Time
	CreatedAt    time.Time
}
