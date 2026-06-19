package agent

import (
	"strings"

	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	toolresultpkg "github.com/wzhejunqiu/ds-code/internal/toolresult"
	"go.uber.org/zap"
)

func (r *Runner) finalizeToolResult(sessionID string, tc llm.ToolCall, formattedBody string) string {
	if r.Tools != nil && r.MCPResults != nil && r.Tools.IsMCPTool(tc.Name) &&
		formattedBody != "" && !isToolErrorBody(formattedBody) {
		path, err := r.MCPResults.Save(sessionID, tc.ID, formattedBody)
		if err == nil {
			max := r.Cfg.Context.ToolResultMaxChars
			if max <= 0 {
				max = 100000
			}
			if len(formattedBody) > max {
				displayPath := toolresultpkg.ShortenSpillPathForHint(path, max)
				hint := toolresultpkg.MCPSavedResultHint(displayPath)
				budget := max - len(hint)
				if budget < 0 {
					budget = 0
				}
				truncated := formattedBody
				if len(truncated) > budget {
					truncated = truncated[:budget]
				}
				return truncated + hint
			}
			return formattedBody
		}
		logging.L().Warn("mcp result spill failed",
			zap.String("session_id", sessionID),
			zap.String("tool", tc.Name),
			zap.Error(err),
		)
	}
	return ctxpkg.TruncateToolResult(formattedBody, r.Cfg)
}

func isToolErrorBody(formatted string) bool {
	inner, isErr := toolresultpkg.UnpackToolBody(formatted)
	if isErr {
		return true
	}
	return strings.HasPrefix(strings.TrimSpace(inner), "error:")
}
