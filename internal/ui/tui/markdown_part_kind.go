package tui

// markdownPartKind splits assistant markdown into renderable segments.
type markdownPartKind int

const (
	markdownPartHeading markdownPartKind = iota // ATX # heading line
	markdownPartBody                            // glamour-rendered markdown body
)
