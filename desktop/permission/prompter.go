package permission

import (
	"sync"

	"github.com/google/uuid"
	desktopbridge "github.com/wzhejunqiu/ds-code/desktop/bridge"
	"github.com/wzhejunqiu/ds-code/internal/permission"
)

// PermissionEmitter sends permission.request envelopes to the UI.
type PermissionEmitter func(payload desktopbridge.PermissionRequestPayload)

// Registry tracks pending permission requests keyed by ID.
type Registry struct {
	mu         sync.Mutex
	pending    map[string]chan bool
	webPending map[string]chan webFetchReply
	emit       PermissionEmitter
}

// NewRegistry creates a permission wait registry.
func NewRegistry(emit PermissionEmitter) *Registry {
	return &Registry{
		pending: make(map[string]chan bool),
		emit:    emit,
	}
}

// Prompter returns a permission.Prompter for write/shell approval (ask mode).
func (r *Registry) Prompter() permission.Prompter {
	return func(tool, summary string) (bool, error) {
		id := uuid.NewString()
		reply := make(chan bool, 1)
		r.register(id, reply)
		r.emit(desktopbridge.PermissionRequestPayload{
			ID: id, Kind: "write_shell", Tool: tool, Summary: summary,
		})
		allowed := <-reply
		return allowed, nil
	}
}

func (r *Registry) register(id string, reply chan bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pending[id] = reply
}

// Resolve completes a pending permission request.
func (r *Registry) Resolve(id string, allow bool) bool {
	r.mu.Lock()
	reply, ok := r.pending[id]
	if ok {
		delete(r.pending, id)
	}
	r.mu.Unlock()
	if !ok {
		return false
	}
	reply <- allow
	return true
}

// DenyAll resolves all pending requests as denied (e.g. on turn cancel).
func (r *Registry) DenyAll() {
	r.mu.Lock()
	pending := r.pending
	webPending := r.webPending
	r.pending = make(map[string]chan bool)
	r.webPending = make(map[string]chan webFetchReply)
	r.mu.Unlock()
	for _, reply := range pending {
		select {
		case reply <- false:
		default:
		}
	}
	for _, reply := range webPending {
		select {
		case reply <- webFetchReply{choice: permission.WebFetchDeny}:
		default:
		}
	}
}
