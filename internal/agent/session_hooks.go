package agent

import "context"

// EndSessionHooks fires SessionEnd hooks for a session if SessionStart was previously fired.
func (r *Runner) EndSessionHooks(ctx context.Context, sessionID string) {
	if r == nil || r.Hooks == nil || sessionID == "" {
		return
	}
	if r.sessionStarted == nil || !r.sessionStarted[sessionID] {
		return
	}
	delete(r.sessionStarted, sessionID)
	r.Hooks.Run(ctx, HookSessionEnd, marshalHookInput(HookInput{SessionID: sessionID}))
}
