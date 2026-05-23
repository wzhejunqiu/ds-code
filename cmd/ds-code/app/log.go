package app

import (
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
)

// LogConfigResolved logs a summary of the loaded configuration at debug level.
func LogConfigResolved(cfg *config.Config) {
	if cfg == nil {
		return
	}
	logging.L().Debug("config resolved",
		zap.String("project_root", cfg.ProjectRoot),
		zap.String("model", cfg.LLM.Model),
		zap.String("permission_mode", cfg.Permission.Mode),
		zap.String("run_mode", cfg.RunMode),
		zap.Int("log_verbosity", cfg.LogVerbosity),
		zap.Bool("allow_log_sensitive_data", cfg.AllowLogSensitiveData),
		zap.Int("mcp_servers", len(cfg.MCP.Servers)),
		zap.Bool("json_output", cfg.JSONOutput),
	)
}
