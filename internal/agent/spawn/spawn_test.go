package spawn_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/read_file"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/write_file"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/glob"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/grep"
)

func testConfig() *config.Config {
	return &config.Config{
		ProjectRoot: "/tmp/test-project",
		LLM: config.LLMConfig{
			Model: "deepseek-v4-pro",
			Subagent: config.SubagentLLMConfig{
				Model:           "",
				ReasoningEffort: "low",
				MaxTurns:        5,
			},
		},
		Agent: config.AgentConfig{MaxTurns: 8},
		Tools: config.ToolsConfig{
			Agent: config.AgentToolConfig{
				ForkEnabled:     true,
				SummaryMaxChars: 0,
			},
		},
	}
}

func testPerm() *permission.Engine {
	return permission.NewEngine("auto", "/tmp/test-project", false)
}

func testRegistry() *tool.Registry {
	reg := tool.NewRegistry()
	reg.Register(&read_file.ReadFileTool{Cfg: testConfig(), Perm: testPerm(), Strict: false})
	reg.Register(&write_file.WriteFileTool{Cfg: testConfig(), Perm: testPerm(), Strict: false})
	reg.Register(&glob.GlobTool{Cfg: testConfig(), Perm: testPerm(), Strict: false})
	reg.Register(&grep.GrepTool{Cfg: testConfig(), Perm: testPerm(), Strict: false})
	return reg
}

// --- Route tests ---

func TestRoute_ForkPath(t *testing.T) {
	cfg := testConfig()
	cfg.Tools.Agent.ForkEnabled = true
	reg := spawn.NewRegistry()
	inv := agent.ToolInvocation{SessionID: "s1", ToolCallID: "tc1"}

	decision, err := spawn.Route(spawn.Params{
		Prompt: "do something",
	}, inv, reg, cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.IsFork {
		t.Error("expected fork when subagent_type omitted and fork enabled + interactive")
	}
}

func TestRoute_ExplicitType(t *testing.T) {
	cfg := testConfig()
	reg := spawn.NewRegistry()
	inv := agent.ToolInvocation{SessionID: "s1", ToolCallID: "tc1"}

	decision, err := spawn.Route(spawn.Params{
		SubagentType: "Explore",
		Prompt:       "find stuff",
	}, inv, reg, cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	if decision.IsFork {
		t.Error("expected non-fork when subagent_type is explicit")
	}
	if decision.Definition.Type != "Explore" {
		t.Errorf("expected Explore, got %s", decision.Definition.Type)
	}
}

func TestRoute_ForceBackground(t *testing.T) {
	cfg := testConfig()
	cfg.Tools.Agent.ForkEnabled = false
	reg := spawn.NewRegistry()
	inv := agent.ToolInvocation{SessionID: "s1", ToolCallID: "tc1"}

	decision, err := spawn.Route(spawn.Params{
		SubagentType: "verification",
		Prompt:       "verify changes",
	}, inv, reg, cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Background {
		t.Error("verification agent should be forced background")
	}
}

func TestRoute_ExplicitBackground(t *testing.T) {
	cfg := testConfig()
	cfg.Tools.Agent.ForkEnabled = false
	reg := spawn.NewRegistry()
	inv := agent.ToolInvocation{SessionID: "s1", ToolCallID: "tc1"}

	decision, err := spawn.Route(spawn.Params{
		SubagentType:    "general-purpose",
		Prompt:          "do work",
		RunInBackground: true,
	}, inv, reg, cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Background {
		t.Error("expected background when run_in_background is true")
	}
}

func TestRoute_UnknownType(t *testing.T) {
	cfg := testConfig()
	reg := spawn.NewRegistry()
	inv := agent.ToolInvocation{SessionID: "s1", ToolCallID: "tc1"}

	_, err := spawn.Route(spawn.Params{
		SubagentType: "nonexistent",
		Prompt:       "do work",
	}, inv, reg, cfg, true)
	if err == nil {
		t.Error("expected error for unknown agent type")
	}
}

func TestRoute_DefaultType(t *testing.T) {
	cfg := testConfig()
	cfg.Tools.Agent.ForkEnabled = false
	reg := spawn.NewRegistry()
	inv := agent.ToolInvocation{SessionID: "s1", ToolCallID: "tc1"}

	decision, err := spawn.Route(spawn.Params{
		Prompt: "do work",
	}, inv, reg, cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Definition.Type != "general-purpose" {
		t.Errorf("expected general-purpose as default, got %s", decision.Definition.Type)
	}
}

func TestRoute_NonInteractiveNoFork(t *testing.T) {
	cfg := testConfig()
	cfg.Tools.Agent.ForkEnabled = true
	reg := spawn.NewRegistry()
	inv := agent.ToolInvocation{SessionID: "s1", ToolCallID: "tc1"}

	decision, err := spawn.Route(spawn.Params{
		Prompt: "do work",
	}, inv, reg, cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	if decision.IsFork {
		t.Error("fork should not be used in non-interactive mode")
	}
}

// --- FilterToolRegistry tests ---

func TestFilterToolRegistry_GlobalBlock(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Register(&read_file.ReadFileTool{Cfg: testConfig(), Perm: testPerm(), Strict: false})
	// Simulate an agent tool — should be blocked globally.
	dummy := &dummyTool{name: "agent", readOnly: true, concurrencySafe: true}
	reg.Register(dummy)

	filtered := spawn.FilterToolRegistry(reg, spawn.AgentTypeDefinition{
		Type:  "general-purpose",
		Tools: []string{"*"},
	}, false)

	if _, ok := filtered.Get("agent"); ok {
		t.Error("agent tool should be globally blocked")
	}
	if _, ok := filtered.Get("read_file"); !ok {
		t.Error("read_file should pass through")
	}
}

func TestFilterToolRegistry_TypeDisallowed(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Register(&read_file.ReadFileTool{Cfg: testConfig(), Perm: testPerm(), Strict: false})
	reg.Register(&write_file.WriteFileTool{Cfg: testConfig(), Perm: testPerm(), Strict: false})

	filtered := spawn.FilterToolRegistry(reg, spawn.AgentTypeDefinition{
		Type:            "Explore",
		Tools:           []string{"*"},
		DisallowedTools: []string{"write_file"},
	}, false)

	if _, ok := filtered.Get("read_file"); !ok {
		t.Error("read_file should be allowed for Explore")
	}
	if _, ok := filtered.Get("write_file"); ok {
		t.Error("write_file should be disallowed for Explore")
	}
}

func TestFilterToolRegistry_AsyncWhitelist(t *testing.T) {
	reg := testRegistry()

	filtered := spawn.FilterToolRegistry(reg, spawn.AgentTypeDefinition{
		Type:  "general-purpose",
		Tools: []string{"*"},
	}, true)

	if _, ok := filtered.Get("write_file"); !ok {
		t.Error("write_file should be in async whitelist")
	}
	if _, ok := filtered.Get("read_file"); !ok {
		t.Error("read_file should be in async whitelist")
	}
}

// --- BuildForkMessages tests ---

func TestBuildForkMessages_BasicStructure(t *testing.T) {
	parentToolCalls := []llm.ToolCall{
		{ID: "call_1", Name: "agent", Arguments: `{"prompt":"do something"}`},
	}

	parentMessages := []llm.Message{
		{Role: role.User, Content: "help me with this"},
		{Role: role.Assistant, Content: "", ToolCalls: parentToolCalls},
	}

	result := spawn.BuildForkMessages(parentMessages, parentToolCalls, "forked directive")
	if len(result) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(result))
	}

	last := result[len(result)-1]
	if last.Role != role.User {
		t.Errorf("expected last message to be user, got %s", last.Role)
	}
	if !strings.Contains(last.Content, "Fork started — processing in background") {
		t.Error("expected fork placeholder in last message")
	}
	if !strings.Contains(last.Content, "[directive:") || !strings.Contains(last.Content, "forked directive") {
		t.Error("expected directive in last message")
	}
}

func TestBuildForkMessages_EmptyHistory(t *testing.T) {
	result := spawn.BuildForkMessages(nil, nil, "directive")
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Role != role.User {
		t.Error("expected user role")
	}
}

// --- NotificationQueue tests ---

func TestNotificationQueue_EnqueueAndDrain(t *testing.T) {
	q := spawn.NewNotificationQueue()
	q.Enqueue(spawn.Notification{AgentID: "a1", Status: "completed"}, spawn.PrioNext)
	q.Enqueue(spawn.Notification{AgentID: "a2", Status: "completed"}, spawn.PrioNext)

	next := q.Drain(spawn.PrioNext)
	if len(next) != 2 {
		t.Errorf("expected 2, got %d", len(next))
	}
	if q.HasPending() {
		t.Error("queue should be empty after drain")
	}
}

func TestNotificationQueue_Dedup(t *testing.T) {
	q := spawn.NewNotificationQueue()
	q.Enqueue(spawn.Notification{AgentID: "a1"}, spawn.PrioNext)
	q.Enqueue(spawn.Notification{AgentID: "a1"}, spawn.PrioNext)
	q.Enqueue(spawn.Notification{AgentID: "a2"}, spawn.PrioNow)

	if q.HasPending() {
		next := q.Drain(spawn.PrioNext)
		if len(next) != 1 {
			t.Errorf("expected 1 (deduped), got %d", len(next))
		}
	}
	now := q.Drain(spawn.PrioNow)
	if len(now) != 1 {
		t.Errorf("expected 1 for PrioNow, got %d", len(now))
	}
}

func TestNotificationQueue_PrioritySeparation(t *testing.T) {
	q := spawn.NewNotificationQueue()
	q.Enqueue(spawn.Notification{AgentID: "a1"}, spawn.PrioNow)
	q.Enqueue(spawn.Notification{AgentID: "a2"}, spawn.PrioNext)
	q.Enqueue(spawn.Notification{AgentID: "a3"}, spawn.PrioLater)

	now := q.Drain(spawn.PrioNow)
	if len(now) != 1 || now[0].AgentID != "a1" {
		t.Error("PrioNow drain failed")
	}
	next := q.Drain(spawn.PrioNext)
	if len(next) != 1 || next[0].AgentID != "a2" {
		t.Error("PrioNext drain failed")
	}
	later := q.Drain(spawn.PrioLater)
	if len(later) != 1 || later[0].AgentID != "a3" {
		t.Error("PrioLater drain failed")
	}
}

func TestNotificationQueue_FormatXML(t *testing.T) {
	n := spawn.Notification{
		AgentID:      "a1",
		ToolUseID:    "tc1",
		OutputFile:   "/tmp/out",
		Status:       "completed",
		Summary:      "done",
		Result:       "all good",
		DurationMS:   1234,
		ToolUseCount: 5,
		Usage:        llm.Usage{PromptTokens: 100, CompletionTokens: 50},
	}
	xml := n.FormatXML()
	if !strings.Contains(xml, "<task-notification>") {
		t.Error("expected task-notification tag")
	}
	if !strings.Contains(xml, "<status>completed</status>") {
		t.Error("expected completed status")
	}
	if !strings.Contains(xml, "a1") {
		t.Error("expected agent ID")
	}
}

func TestNotificationQueue_FormatXML_WithWorktree(t *testing.T) {
	n := spawn.Notification{
		AgentID:        "a1",
		Status:         "completed",
		WorktreePath:   "/tmp/wt",
		WorktreeBranch: "wt-branch",
	}
	xml := n.FormatXML()
	if !strings.Contains(xml, "<worktree>") {
		t.Error("expected worktree info")
	}
}

// --- ResolveModel tests ---

func TestResolveModel_ParamsOverride(t *testing.T) {
	cfg := testConfig()
	result := spawn.ResolveModel("sonnet", "inherit", cfg)
	if result != cfg.LLM.Model {
		t.Errorf("params model alias should resolve to main model, got %s", result)
	}
}

func TestResolveModel_DefOverride(t *testing.T) {
	cfg := testConfig()
	result := spawn.ResolveModel("", "specific-model-v2", cfg)
	if result != "specific-model-v2" {
		t.Errorf("definition model should take priority, got %s", result)
	}
}

func TestResolveModel_SubagentConfig(t *testing.T) {
	cfg := testConfig()
	cfg.LLM.Subagent.Model = "sub-model"
	result := spawn.ResolveModel("", "", cfg)
	if result != "sub-model" {
		t.Errorf("expected sub-model from config, got %s", result)
	}
}

func TestResolveModel_FallbackToMain(t *testing.T) {
	cfg := testConfig()
	cfg.LLM.Subagent.Model = ""
	result := spawn.ResolveModel("", "", cfg)
	if result != cfg.LLM.Model {
		t.Errorf("expected fallback to main model, got %s", result)
	}
}

// --- Registry tests ---

func TestRegistry_ListTypesExcludesFork(t *testing.T) {
	reg := spawn.NewRegistry()
	types := reg.ListTypes()
	for _, typ := range types {
		if typ == "fork" {
			t.Error("ListTypes should exclude synthetic fork type")
		}
	}
	if len(types) < 4 {
		t.Errorf("expected at least 4 built-in types, got %d", len(types))
	}
}

func TestRegistry_ResolveEmptyDefaults(t *testing.T) {
	reg := spawn.NewRegistry()
	def, err := reg.Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if def.Type != "general-purpose" {
		t.Errorf("expected general-purpose default, got %s", def.Type)
	}
}

func TestRegistry_ExploreIsReadOnly(t *testing.T) {
	reg := spawn.NewRegistry()
	def, err := reg.Resolve("Explore")
	if err != nil {
		t.Fatal(err)
	}
	if !spawn.IsReadOnly(def) {
		t.Error("Explore should be read-only")
	}
}

// --- IsReadOnly tests ---

func TestIsReadOnly_ExplicitFlag(t *testing.T) {
	def := spawn.AgentTypeDefinition{ReadOnly: true}
	if !spawn.IsReadOnly(def) {
		t.Error("expected read-only")
	}
}

func TestIsReadOnly_PermissionMode(t *testing.T) {
	def := spawn.AgentTypeDefinition{PermissionMode: "readonly"}
	if !spawn.IsReadOnly(def) {
		t.Error("readonly permission mode should be read-only")
	}
}

func TestIsReadOnly_NotReadOnly(t *testing.T) {
	def := spawn.AgentTypeDefinition{}
	if spawn.IsReadOnly(def) {
		t.Error("should not be read-only")
	}
}

// --- Context helpers ---

func TestWithQuerySource(t *testing.T) {
	ctx := spawn.WithQuerySource(context.Background(), spawn.QuerySourceExplore)
	if got := spawn.QuerySourceFromContext(ctx); got != spawn.QuerySourceExplore {
		t.Errorf("expected QuerySourceExplore, got %s", got)
	}
}

func TestQuerySourceFromContext_Default(t *testing.T) {
	got := spawn.QuerySourceFromContext(context.Background())
	if got != spawn.QuerySourceAgent {
		t.Errorf("expected default QuerySourceAgent, got %s", got)
	}
}

func TestWithRenderedSystem(t *testing.T) {
	ctx := agent.WithRenderedSystem(context.Background(), "system prompt here")
	if got := agent.RenderedSystemFromContext(ctx); got != "system prompt here" {
		t.Errorf("expected system prompt, got %s", got)
	}
}

func TestRenderedSystemFromContext_Empty(t *testing.T) {
	if got := agent.RenderedSystemFromContext(context.Background()); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

// --- dummyTool for testing ---

type dummyTool struct {
	name           string
	readOnly       bool
	concurrencySafe bool
}

func (d *dummyTool) Name() string                     { return d.name }
func (d *dummyTool) Description() string              { return "dummy" }
func (d *dummyTool) Schema() map[string]any           { return map[string]any{} }
func (d *dummyTool) Execute(_ context.Context, _ json.RawMessage) (string, error) { return "", nil }
func (d *dummyTool) PermissionLevel() permission.Level { return permission.LevelLow }
func (d *dummyTool) IsReadOnly() bool                 { return d.readOnly }
func (d *dummyTool) IsConcurrencySafe() bool          { return d.concurrencySafe }
