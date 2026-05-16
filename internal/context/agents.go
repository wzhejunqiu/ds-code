package context

import (
	"os"
	"path/filepath"
)

// LoadAgentsMD reads AGENTS.md from project root if present.
func LoadAgentsMD(projectRoot string) (string, error) {
	path := filepath.Join(projectRoot, "AGENTS.md")
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}
