package state

import (
	"context"
	"time"

	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/ui/slash"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/hejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/hejunqiu/ds-code/internal/ui/tui/subagent"
)

// State holds mutable TUI model fields shared across model subpackages.
type State struct {
	Deps      *deps.Deps
	SessionID string
	Width     int
	Height    int

	Chat      []chat.Block
	ToolLines []string
	ToolOpen  bool

	// MainChat is the primary session transcript; Chat is what the viewport shows.
	MainChat      []chat.Block
	MainToolLines []string

	Subagents         subagent.Registry
	SubagentNav       SubagentNav
	ViewingSubagentID string

	Overlay     OverlayKind
	OverlayText string

	Complete          []slash.Command
	CompleteFilterKey string

	ResumeSessions  []session.Summary
	ResumeFilter    string
	ResumeFilterSeq uint64
	ResumePending   bool

	Prompt *permission.PromptRequest

	Running            bool
	TurnCancel         context.CancelFunc
	TurnEscPending     bool
	ReasoningAll       bool
	ToolDetailsVisible bool

	StatusRight string
	ErrLine     string

	HeaderSession session.Session
	HasSession    bool

	ExitConfirmPending bool
	ExitConfirmKey     string
	ExitConfirmArmedAt time.Time
}
