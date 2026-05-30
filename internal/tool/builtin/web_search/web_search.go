package web_search

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
)

// WebSearchTool is a placeholder until a search provider is integrated.
type WebSearchTool struct {
	Cfg    *config.Config
	Strict bool
}

func (t *WebSearchTool) Name() string { return tool.NameWebSearch.String() }

func (t *WebSearchTool) IsReadOnly() bool        { return true }
func (t *WebSearchTool) IsConcurrencySafe() bool { return true }

func (t *WebSearchTool) Description() string { return DescWebSearch }

func (t *WebSearchTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"query": map[string]any{"type": "string", "description": SchemaQuery},
	}, []string{"query"}, t.Strict)
}

func (t *WebSearchTool) PermissionLevel() permission.Level { return permission.LevelMedium }

func (t *WebSearchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if !t.Cfg.Web.SearchEnabled {
		return "", fmt.Errorf("%s", ErrDisabled)
	}
	var in struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if in.Query == "" {
		return "", fmt.Errorf("%s", builtin.ErrQueryRequired)
	}
	return "", fmt.Errorf("%s", ErrNotConfigured)
}
