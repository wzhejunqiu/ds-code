package state

import (
	"context"
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/slash"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/subagent"
)

// TCaseItem is one harness integration scenario row (name + short description).
type TCaseItem struct {
	Name string
	Desc string
}

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

	TCaseItems []TCaseItem // harness: /tcase picker rows (name + description)

	Prompt *permission.PromptRequest

	Running            bool
	TurnWG             sync.WaitGroup
	TurnCancel         context.CancelFunc
	TurnEscPending     bool
	ReasoningAll       bool
	ToolDetailsVisible bool

	StatusRight      string
	ErrLine          string
	SensitiveLogWarn string

	HeaderSession session.Session
	HeaderCostCNY float64
	HasSession    bool

	ExitConfirmPending bool
	ExitConfirmKey     string
	ExitConfirmArmedAt time.Time
}
