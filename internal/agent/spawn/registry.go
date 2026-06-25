package spawn

import (
	"fmt"
	"sort"

	"github.com/wzhejunqiu/ds-code/internal/agent/spawn/builtin"
)

// Registry holds all available agent types.
type Registry struct {
	types map[AgentType]AgentTypeDefinition
}

// NewRegistry creates a registry with built-in agent types from the builtin package.
func NewRegistry() *Registry {
	r := &Registry{types: make(map[AgentType]AgentTypeDefinition)}
	for _, def := range builtin.All() {
		r.Register(def)
	}
	return r
}

// Register adds or overwrites an agent type.
func (r *Registry) Register(def AgentTypeDefinition) {
	r.types[def.Type] = def
}

// Resolve returns the definition for a type name. Empty string defaults to general-purpose.
func (r *Registry) Resolve(typeName string) (AgentTypeDefinition, error) {
	t := AgentTypeGeneralPurpose
	if typeName != "" {
		var err error
		t, err = ParseAgentType(typeName)
		if err != nil {
			return AgentTypeDefinition{}, err
		}
	}
	def, ok := r.types[t]
	if !ok {
		return AgentTypeDefinition{}, fmt.Errorf("unknown agent type: %s", typeName)
	}
	return def, nil
}

// ListTypes returns sorted type names for schema enum generation (excludes synthetic fork).
func (r *Registry) ListTypes() []string {
	names := make([]string, 0, len(r.types))
	for n := range r.types {
		if n == AgentTypeFork {
			continue
		}
		names = append(names, n.String())
	}
	sort.Strings(names)
	return names
}

// ListToolTypes returns agent types exposed in the LLM agent tool JSON schema.
func (r *Registry) ListToolTypes() []string {
	names := make([]string, 0, 2)
	for _, t := range []AgentType{AgentTypeGeneralPurpose, AgentTypeExplore} {
		if _, ok := r.types[t]; ok {
			names = append(names, t.String())
		}
	}
	sort.Strings(names)
	return names
}

// IsReadOnly returns true if the agent type is fully read-only.
func IsReadOnly(def AgentTypeDefinition) bool {
	return def.ReadOnly || def.PermissionMode == AgentPermModeReadonly
}
