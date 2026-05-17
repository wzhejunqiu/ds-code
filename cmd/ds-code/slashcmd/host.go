package slashcmd

import "context"

// Host supplies app-level operations that slash handlers cannot perform alone.
type Host interface {
	SetRunMode(ctx context.Context, env *Env, mode string) error
}
