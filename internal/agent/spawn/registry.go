package spawn

import (
	"fmt"
	"sort"
)

// AgentTypeDefinition describes a built-in agent type.
type AgentTypeDefinition struct {
	Type            string // e.g. "general-purpose", "Explore", "Plan", "verification"
	Description     string // when-to-use summary shown in tool schema
	Tools           []string // tool pool spec (["*"] = all allowed)
	DisallowedTools []string // tools forbidden for this type
	Model           string // default model, "inherit" to use parent
	ForceBackground bool   // always run async (verification)
	PermissionMode  string // "inherit", "readonly", "bubble"
	ReadOnly        bool   // entire agent is read-only (Explore, Plan)
	OmitHeavyRules  bool   // skip AGENTS.md / rules injection
}

// Registry holds all available agent types.
type Registry struct {
	types map[string]AgentTypeDefinition
}

// NewRegistry creates a registry with the 4 built-in types.
func NewRegistry() *Registry {
	r := &Registry{types: make(map[string]AgentTypeDefinition)}
	r.registerBuiltins()
	return r
}

func (r *Registry) registerBuiltins() {
	r.Register(AgentTypeDefinition{
		Type:        "general-purpose",
		Description: "全能子代理，可读写文件、执行 shell、使用全部工具。适用于多步骤复杂任务。",
		Tools:       []string{"*"},
		Model:       "inherit",
		DisallowedTools: []string{"agent"},
	})
	r.Register(AgentTypeDefinition{
		Type:        "Explore",
		Description: "快速只读探索代理，用于搜索代码库、查找文件、阅读代码。不可写文件。",
		Tools:       []string{"*"},
		DisallowedTools: []string{"agent", "write_file", "apply_patch"},
		Model:       "inherit",
		ReadOnly:    true,
		OmitHeavyRules: true,
	})
	r.Register(AgentTypeDefinition{
		Type:        "Plan",
		Description: "只读架构规划代理，分析代码结构并输出实施方案。不可写文件。",
		Tools:       []string{"*"},
		DisallowedTools: []string{"agent", "write_file", "apply_patch"},
		Model:       "inherit",
		ReadOnly:    true,
		OmitHeavyRules: true,
	})
	r.Register(AgentTypeDefinition{
		Type:        "verification",
		Description: "对抗性验证代理，独立检查代码变更的正确性和安全性。后台运行，以 VERDICT 结束。",
		Tools:       []string{"*"},
		DisallowedTools: []string{"agent", "write_file", "apply_patch"},
		Model:       "inherit",
		ForceBackground: true,
		ReadOnly:    true,
	})
	// Fork is a synthetic type used when subagent_type is omitted and fork is enabled.
	// It uses the parent's exact tool pool and rendered system prompt.
	r.Register(AgentTypeDefinition{
		Type:            "fork",
		Description:     "Fork 子代理，继承父会话的工具池和 system prompt，共享 prompt cache 前缀。",
		Tools:           []string{"*"},
		DisallowedTools: []string{"agent"},
		Model:           "inherit",
		PermissionMode:  "bubble",
	})
}

// Register adds or overwrites an agent type.
func (r *Registry) Register(def AgentTypeDefinition) {
	r.types[def.Type] = def
}

// Resolve returns the definition for a type name. Empty string defaults to general-purpose.
func (r *Registry) Resolve(typeName string) (AgentTypeDefinition, error) {
	if typeName == "" {
		typeName = "general-purpose"
	}
	def, ok := r.types[typeName]
	if !ok {
		return AgentTypeDefinition{}, fmt.Errorf("unknown agent type: %s", typeName)
	}
	return def, nil
}

// ListTypes returns sorted type names for schema enum generation (excludes synthetic fork).
func (r *Registry) ListTypes() []string {
	names := make([]string, 0, len(r.types))
	for n := range r.types {
		if n == "fork" {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// IsReadOnly returns true if the agent type is fully read-only.
func IsReadOnly(def AgentTypeDefinition) bool {
	return def.ReadOnly || def.PermissionMode == "readonly"
}
