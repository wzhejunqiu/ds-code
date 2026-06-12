package checkpoint

import "time"

// FileState captures a workspace file before a write tool runs.
type FileState struct {
	RelPath string `json:"rel_path"`
	Existed bool   `json:"existed"`
	Content []byte `json:"content,omitempty"`
}

// Record is a restorable checkpoint for one write operation.
type Record struct {
	ID        int         `json:"id"`
	SessionID string      `json:"session_id"`
	Tool      string      `json:"tool"`
	Files     []FileState `json:"files"`
	Patch     string      `json:"patch,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

// Meta is a lightweight list entry.
type Meta struct {
	ID        int       `json:"id"`
	Tool      string    `json:"tool"`
	Files     []string  `json:"files"`
	CreatedAt time.Time `json:"created_at"`
}
