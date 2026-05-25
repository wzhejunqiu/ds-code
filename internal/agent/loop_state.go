package agent

// LoopPhase tracks which stage of the agent loop is executing.
type LoopPhase int

const (
	PhasePrepare  LoopPhase = iota + 1 // building API context, snip, meta reminder
	PhaseLLM                            // streaming LLM call
	PhaseDecide                         // terminal vs tools decision
	PhaseTools                          // tool orchestration
	PhaseUpdate                         // append messages, increment round
)

// Transition records why the loop advanced to the next iteration.
type Transition string

const (
	TransNextTurn             Transition = "next_turn"
	TransCompactRetry         Transition = "compact_retry"
	TransSnipRetry            Transition = "snip_retry"
	TransMaxTokensEscalate    Transition = "max_tokens_escalate"
	TransOutputRecovery       Transition = "output_recovery"
	TransNetworkRetry         Transition = "network_retry"
	TransModelFallback        Transition = "model_fallback"
	TransRateLimitRetry       Transition = "rate_limit_retry"
	TransCompleted            Transition = "completed"
	TransMaxTurns             Transition = "max_turns"
	TransAborted              Transition = "aborted"
)

// LoopState carries mutable recovery state across sub-rounds.
type LoopState struct {
	Phase               LoopPhase
	Transition          Transition
	Round               int
	CompactRetried      bool
	SnipRetried         bool
	MaxTokensEscalated  bool
	OutputRecoveryCount int
	NetworkRetryCount   int
	FallbackTried       bool
}
