package agent

import (
	"encoding/json"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestAgentTool_schema(t *testing.T) {
	cfg := &config.Config{
		ProjectRoot: t.TempDir(),
		LLM:         config.LLMConfig{Model: "deepseek-v4-pro"},
		Tools:       config.ToolsConfig{Agent: config.AgentToolConfig{MaxParallel: 3}},
	}
	perm := permission.NewEngine("readonly", cfg.ProjectRoot, false)
	reg := tool.NewRegistry()
	at := NewAgentTool(cfg, perm, nil, false, subagentstore.NewMemoryStore(), reg)

	schema := at.Schema()
	if schema["type"] != "object" {
		t.Fatalf("expected object schema, got %v", schema["type"])
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("missing properties")
	}
	for _, key := range []string{"description", "prompt", "subagent_type"} {
		if _, ok := props[key]; !ok {
			t.Fatalf("missing property %s", key)
		}
	}
	reqSlice, ok := schema["required"].([]string)
	if !ok {
		if anyReq, ok2 := schema["required"].([]any); ok2 {
			reqSlice = make([]string, len(anyReq))
			for i, v := range anyReq {
				reqSlice[i], _ = v.(string)
			}
		} else {
			t.Fatalf("required: %T", schema["required"])
		}
	}
	if len(reqSlice) < 2 {
		t.Fatalf("expected required fields, got %v", reqSlice)
	}
	enum, ok := props["subagent_type"].(map[string]any)["enum"].([]any)
	if !ok || len(enum) != 2 {
		t.Fatalf("expected 2 subagent_type enum values, got %v", props["subagent_type"])
	}
	for _, v := range enum {
		switch v {
		case "general-purpose", "Explore":
		default:
			t.Fatalf("unexpected subagent_type enum value %v", v)
		}
	}
	for _, forbidden := range []string{"fork", "Plan", "verification"} {
		for _, v := range enum {
			if v == forbidden {
				t.Fatalf("%s should not appear in schema enum", forbidden)
			}
		}
	}
	if _, ok := props["model"]; ok {
		t.Fatal("model should not appear in agent tool schema")
	}
	if _, ok := props["isolation"]; ok {
		t.Fatal("isolation should not appear in agent tool schema")
	}
}

func TestAgentTool_execute_requires_parent(t *testing.T) {
	cfg := &config.Config{ProjectRoot: t.TempDir()}
	perm := permission.NewEngine("readonly", cfg.ProjectRoot, false)
	reg := tool.NewRegistry()
	at := NewAgentTool(cfg, perm, nil, false, subagentstore.NewMemoryStore(), reg)
	_, err := at.Execute(t.Context(), json.RawMessage(`{"description":"x","prompt":"y"}`))
	if err == nil {
		t.Fatal("expected error without parent invocation")
	}
}
