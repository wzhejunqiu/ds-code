package deepseek

import (
	"fmt"
	"os"
	"path/filepath"
)

// AssetsDir is the repo path to bundled tokenizer files (tokenizer.json, tokenizer_config.json).
const AssetsDir = "internal/assets/deepseek-v4"

const tokenizerFile = "tokenizer.json"

// resolveTokenizerFile returns an absolute path to tokenizer.json under dir.
// Empty dir uses the embedded default (see New).
func resolveTokenizerFile(dir string) (string, bool, error) {
	if dir == "" {
		return "", false, nil
	}
	base, err := filepath.Abs(dir)
	if err != nil {
		return "", false, err
	}
	path := filepath.Join(base, tokenizerFile)
	if _, err := os.Stat(path); err != nil {
		return "", false, fmt.Errorf("deepseek tokenizer: missing %s: %w", path, err)
	}
	return path, true, nil
}
