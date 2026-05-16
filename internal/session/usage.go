package session

// UsageSnapshot is cumulative session usage for status bar and /context.
type UsageSnapshot struct {
	PromptTokensTotal         int64
	CompletionTokensTotal     int64
	PromptCacheHitTokensTotal int64
	Billed                    int
}

// UsageSnapshotFromSession builds a snapshot from session totals.
func UsageSnapshotFromSession(s Session) UsageSnapshot {
	return UsageSnapshot{
		PromptTokensTotal:         s.PromptTokensTotal,
		CompletionTokensTotal:     s.CompletionTokensTotal,
		PromptCacheHitTokensTotal: s.PromptCacheHitTokensTotal,
		Billed:                    BilledTokens(s),
	}
}
