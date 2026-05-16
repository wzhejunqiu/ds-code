package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// WebSearchTool is a placeholder until a search provider is integrated.
type WebSearchTool struct {
	Cfg    *config.Config
	Strict bool
}

func (t *WebSearchTool) Name() string { return "web_search" }

func (t *WebSearchTool) Description() string {
	return "Search the web (requires web.search_enabled in config)."
}

func (t *WebSearchTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"query": map[string]any{"type": "string", "description": "Search query"},
	}, []string{"query"}, t.Strict)
}

func (t *WebSearchTool) PermissionLevel() permission.Level { return permission.LevelMedium }

func (t *WebSearchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if !t.Cfg.Web.SearchEnabled {
		return "", fmt.Errorf("web_search is disabled (set web.search_enabled: true)")
	}
	var in struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if in.Query == "" {
		return "", fmt.Errorf("query is required")
	}
	return "", fmt.Errorf("web_search provider not configured; use web_fetch with a known URL")
}
