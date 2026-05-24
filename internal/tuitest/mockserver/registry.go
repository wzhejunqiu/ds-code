//go:build tuitest

package mockserver

import (
	"sync"

	"github.com/wzhejunqiu/ds-code/internal/tuitest/scenarios"
)

// Registry holds scenarios and active turn state for the mock LLM server.
type Registry struct {
	mu          sync.Mutex
	byName      map[string]*scenarios.Scenario
	active      string
	turnIndex   int
	contextFail bool // error-context: inject one context-too-long response first
}

// NewRegistry returns a registry preloaded with built-in scenarios.
func NewRegistry() *Registry {
	r := &Registry{byName: make(map[string]*scenarios.Scenario)}
	for _, sc := range scenarios.All() {
		r.byName[sc.Name] = sc
	}
	return r
}

// SetActive selects a scenario and resets the turn counter.
func (r *Registry) SetActive(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byName[name]; !ok {
		return errUnknownScenario{name}
	}
	r.active = name
	r.turnIndex = 0
	r.contextFail = name == "error-context"
	return nil
}

// ActiveName returns the current scenario name.
func (r *Registry) ActiveName() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active
}

// List returns scenario names in stable order.
func (r *Registry) List() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	names := make([]string, 0, len(r.byName))
	for _, sc := range scenarios.All() {
		if _, ok := r.byName[sc.Name]; ok {
			names = append(names, sc.Name)
		}
	}
	return names
}

// Get returns a scenario by name.
func (r *Registry) Get(name string) (*scenarios.Scenario, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sc, ok := r.byName[name]
	return sc, ok
}

// NextTurn consumes the next turn for the active scenario.
func (r *Registry) NextTurn() (*scenarios.Turn, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active == "" {
		return nil, false
	}
	sc := r.byName[r.active]
	if sc == nil || r.turnIndex >= len(sc.Turns) {
		return &scenarios.Turn{
			Chunks:       []scenarios.StreamChunk{{Content: "done"}},
			FinishReason: "stop",
		}, true
	}
	if r.contextFail {
		r.contextFail = false
		return &scenarios.Turn{
			HTTPStatus: 400,
			ErrBody:    `{"error":{"message":"context length exceeded","type":"invalid_request"}}`,
		}, true
	}

	t := sc.Turns[r.turnIndex]
	r.turnIndex++
	return &t, true
}

type errUnknownScenario struct{ name string }

func (e errUnknownScenario) Error() string {
	return "tuitest: unknown scenario: " + e.name
}
