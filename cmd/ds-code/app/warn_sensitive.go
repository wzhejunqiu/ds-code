package app

import (
	"io"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
)

// MaybeWarnSensitiveLog prints a red stderr warning when sensitive debug logging is enabled.
func MaybeWarnSensitiveLog(cfg *config.Config, w io.Writer) {
	if cfg != nil && cfg.AllowLogSensitiveData {
		logging.PrintSensitiveDataWarning(w)
	}
}
