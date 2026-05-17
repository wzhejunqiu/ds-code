package limits

const (
	// ContextWindowTokens is the DeepSeek V4 context window (1 MiB).
	ContextWindowTokens = 1_048_576
	// MaxOutputTokens is the maximum completion tokens per request.
	MaxOutputTokens = 393_216
)
