package chattool

import "time"

// Block is the input for rendering a tool row in the chat transcript.
type Block struct {
	Name, Args, Command, Result string
	Running, Error, Expanded    bool
	TimeoutDeadline             time.Time
}
