package shelljobs

import "time"

// Status of a background shell job.
const (
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusKilled    = "killed"
)

// Job describes a background shell command.
type Job struct {
	ID         string     `json:"id"`
	Command    string     `json:"command"`
	PID        int        `json:"pid"`
	Status     string     `json:"status"`
	ExitCode   *int       `json:"exit_code,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	StdoutPath string     `json:"stdout_path"`
	StderrPath string     `json:"stderr_path"`
}
