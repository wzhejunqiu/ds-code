package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	envDSCodeDeepSeek = "DS_CODE_DEEPSEEK_API_KEY"
	envDeepSeek       = "DEEPSEEK_API_KEY"
)

// LoadAPIKey reads the DeepSeek API key from environment variables only.
func LoadAPIKey() (string, error) {
	if k := strings.TrimSpace(os.Getenv(envDSCodeDeepSeek)); k != "" {
		return k, nil
	}
	if k := strings.TrimSpace(os.Getenv(envDeepSeek)); k != "" {
		return k, nil
	}
	return "", fmt.Errorf(
		"missing DeepSeek API key: set %s or %s",
		envDSCodeDeepSeek,
		envDeepSeek,
	)
}
