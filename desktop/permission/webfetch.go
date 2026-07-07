package permission

import (
	"github.com/google/uuid"
	desktopbridge "github.com/wzhejunqiu/ds-code/desktop/bridge"
	"github.com/wzhejunqiu/ds-code/internal/permission"
)

type webFetchReply struct {
	choice permission.WebFetchChoice
}

// WebFetchPrompter returns a permission.WebFetchPrompter for web_fetch approval.
func (r *Registry) WebFetchPrompter() permission.WebFetchPrompter {
	return func(host, rawURL string) (permission.WebFetchChoice, error) {
		id := uuid.NewString()
		reply := make(chan webFetchReply, 1)
		r.registerWeb(id, reply)
		r.emit(desktopbridge.PermissionRequestPayload{
			ID:   id,
			Kind: "web_fetch",
			Tool: "web_fetch",
			Host: host,
			URL:  rawURL,
		})
		res := <-reply
		return res.choice, nil
	}
}

func (r *Registry) registerWeb(id string, reply chan webFetchReply) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.webPending == nil {
		r.webPending = make(map[string]chan webFetchReply)
	}
	r.webPending[id] = reply
}

// ResolveWebFetch completes a pending web_fetch permission request.
func (r *Registry) ResolveWebFetch(id string, choice permission.WebFetchChoice) bool {
	r.mu.Lock()
	reply, ok := r.webPending[id]
	if ok {
		delete(r.webPending, id)
	}
	r.mu.Unlock()
	if !ok {
		return false
	}
	reply <- webFetchReply{choice: choice}
	return true
}

// ResolveChoice handles unified permission resolution from the UI.
// choice: "allow" | "deny" for write_shell; "allow_once" | "allow_always" | "deny" for web_fetch.
func (r *Registry) ResolveChoice(id, choice string) bool {
	r.mu.Lock()
	_, webOk := r.webPending[id]
	_, shellOk := r.pending[id]
	r.mu.Unlock()
	if webOk {
		switch choice {
		case "allow_once":
			return r.ResolveWebFetch(id, permission.WebFetchAllowOnce)
		case "allow_always":
			return r.ResolveWebFetch(id, permission.WebFetchAllowAlways)
		default:
			return r.ResolveWebFetch(id, permission.WebFetchDeny)
		}
	}
	if shellOk {
		return r.Resolve(id, choice == "allow")
	}
	return false
}
