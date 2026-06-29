package chat

import "github.com/wzhejunqiu/ds-code/internal/version"

import "time"

// Block is one row in the in-memory transcript (rendered by Render).
type Block struct {
	Role                Role
	Content             string
	Reasoning           string
	ReasoningOpen       bool
	ReasoningStartedAt  time.Time
	ReasoningEndedAt    time.Time
	PlanningStartedAt   time.Time
	ReasoningDuration   time.Duration
	TurnDuration        time.Duration
	Streaming           bool
	ToolName            string
	ToolCallID          string
	ToolArgs            string
	ToolCommand         string
	ToolResult          string
	ToolRunning         bool
	ToolError           bool
	ToolExpanded        bool
	ToolTimeoutDeadline time.Time
}

// AppendContent appends assistant/user visible text.
func (b *Block) AppendContent(s string) {
	b.Content += s
}

// AppendReasoning appends thinking trace text.
func (b *Block) AppendReasoning(s string) {
	b.Reasoning += s
}

// FinalizeReasoning closes the thinking phase for this assistant segment.
func (b *Block) FinalizeReasoning(at time.Time) {
	if b.Role != RoleAssistant || b.ReasoningStartedAt.IsZero() || !b.ReasoningEndedAt.IsZero() {
		return
	}
	b.ReasoningEndedAt = at
	if b.ReasoningDuration == 0 {
		d := at.Sub(b.ReasoningStartedAt)
		if d > 0 {
			b.ReasoningDuration = d
		}
	}
}

const (
	userPrompt      = "> "
	assistantBullet = "● "

	// UserPrompt is the visible prefix for user messages (exported for tests).
	UserPrompt = userPrompt
	// AssistantBullet is the visible prefix for assistant messages (exported for tests).
	AssistantBullet = assistantBullet
	planningBullet  = "◦ "
	planningLabel   = "规划下一步行动"
	interruptBullet = "⏹ "
	interruptLabel  = "Turn cancelled (Esc)"
)

// InterruptSessionMarker is stored as a system row so /resume restores the marker.
func InterruptSessionMarker() string {
	return version.SystemPrefix + interruptLabel
}

// InterruptLabel is the human-readable interrupt line shown in the transcript.
func InterruptLabel() string {
	return interruptLabel
}
