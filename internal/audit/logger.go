package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Logger appends JSONL audit records (S10).
type Logger struct {
	path string
	mu   sync.Mutex
}

// NewLogger creates an audit logger at the fixed project audit.jsonl path.
func NewLogger(path string) *Logger {
	return &Logger{path: path}
}

type entry struct {
	Time     string `json:"time"`
	Tool     string `json:"tool"`
	ArgsHash string `json:"args_hash"`
}

// Log records a tool invocation (args hashed, not stored verbatim).
func (l *Logger) Log(tool string, args []byte) error {
	if l == nil || l.path == "" {
		return nil
	}
	sum := sha256.Sum256(args)
	e := entry{
		Time:     time.Now().UTC().Format(time.RFC3339),
		Tool:     tool,
		ArgsHash: hex.EncodeToString(sum[:]),
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("audit log: %w", err)
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}
