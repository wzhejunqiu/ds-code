//go:build tuitest

package tuitest

import (
	"os"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

const harnessAPIKey = "sk-tui-test-mock"

// EnsureHarnessAPIKey sets DS_CODE_DEEPSEEK_API_KEY when no key env is present,
// then loads the key via the production LoadAPIKey path.
func EnsureHarnessAPIKey() (string, error) {
	if strings.TrimSpace(os.Getenv("DS_CODE_DEEPSEEK_API_KEY")) == "" &&
		strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")) == "" {
		_ = os.Setenv("DS_CODE_DEEPSEEK_API_KEY", harnessAPIKey)
	}
	return config.LoadAPIKey()
}
