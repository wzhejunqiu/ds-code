package spawn

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

// ResultStatus is the outward-facing completion label for tool returns and notifications.
type ResultStatus string

const (
	ResultCompleted ResultStatus = "completed"
	ResultFailed    ResultStatus = "failed"
	ResultKilled    ResultStatus = "killed"
)

// String returns the wire-format status label.
func (s ResultStatus) String() string {
	return string(s)
}

// resultStatusFromStore maps a terminal persisted run status to the outward-facing label.
func resultStatusFromStore(status subagentstore.Status) (ResultStatus, error) {
	switch status {
	case subagentstore.StatusCompleted:
		return ResultCompleted, nil
	case subagentstore.StatusError:
		return ResultFailed, nil
	case subagentstore.StatusKilled:
		return ResultKilled, nil
	case subagentstore.StatusRunning:
		return "", fmt.Errorf("spawn: running status has no result label")
	default:
		return "", fmt.Errorf("spawn: unknown subagent status %q", status)
	}
}

// NotificationPriority controls when a notification is drained into the main conversation.
type NotificationPriority int

const (
	PrioNow   NotificationPriority = iota // injected before next user message
	PrioNext                                 // drained at start of next RunTurn
	PrioLater                                // drained when idle
)

// Notification is a completion/failure/kill notice for an async agent.
type Notification struct {
	AgentID        string
	ToolUseID      string
	OutputFile     string
	Status         ResultStatus // completed | failed | killed
	Summary        string
	Result         string // inline body when not spilled (Format uses when OutputFile empty)
	Usage          llm.Usage
	DurationMS     int64
	ToolUseCount   int
	WorktreePath   string
	WorktreeBranch string
}

// notificationPriority picks PrioLater during an active parent turn, else PrioNow.
func notificationPriority(ctx context.Context) NotificationPriority {
	if agent.InActiveTurn(ctx) {
		return PrioLater
	}
	return PrioNow
}

// Format renders the notification as XML for LLM consumption (no usage block).
func (n Notification) Format() string {
	var b strings.Builder
	b.WriteString("<task-notification>\n")
	fmt.Fprintf(&b, "  <task-id>%s</task-id>\n", xmlEscapeText(n.AgentID))
	fmt.Fprintf(&b, "  <tool-use-id>%s</tool-use-id>\n", xmlEscapeText(n.ToolUseID))
	if n.OutputFile != "" {
		fmt.Fprintf(&b, "  <output-file>%s</output-file>\n", xmlEscapeText(n.OutputFile))
	}
	fmt.Fprintf(&b, "  <status>%s</status>\n", xmlEscapeText(n.Status.String()))
	fmt.Fprintf(&b, "  <summary>%s</summary>\n", xmlEscapeText(n.Summary))
	if n.OutputFile == "" && n.Result != "" {
		fmt.Fprintf(&b, "  <result>%s</result>\n", xmlEscapeText(n.Result))
	}
	if n.WorktreePath != "" {
		b.WriteString("  <worktree>\n")
		fmt.Fprintf(&b, "    <worktreePath>%s</worktreePath>\n", xmlEscapeText(n.WorktreePath))
		fmt.Fprintf(&b, "    <worktreeBranch>%s</worktreeBranch>\n", xmlEscapeText(n.WorktreeBranch))
		b.WriteString("  </worktree>\n")
	}
	b.WriteString("</task-notification>")
	return b.String()
}

// FormatXML is an alias for Format (legacy name).
func (n Notification) FormatXML() string {
	return n.Format()
}

// NotificationQueue holds pending agent completion notices with dedup.
type NotificationQueue struct {
	mu       sync.Mutex
	queues   map[NotificationPriority][]Notification
	notified map[string]bool
}

// NewNotificationQueue creates an empty queue.
func NewNotificationQueue() *NotificationQueue {
	return &NotificationQueue{
		queues:   make(map[NotificationPriority][]Notification),
		notified: make(map[string]bool),
	}
}

// Enqueue adds a notification at the given priority. Duplicates are silently dropped.
func (q *NotificationQueue) Enqueue(n Notification, prio NotificationPriority) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.notified[n.AgentID] {
		return
	}
	q.notified[n.AgentID] = true
	q.queues[prio] = append(q.queues[prio], n)
}

// Drain returns and removes all notifications at the given priority.
func (q *NotificationQueue) Drain(prio NotificationPriority) []Notification {
	q.mu.Lock()
	defer q.mu.Unlock()
	result := q.queues[prio]
	delete(q.queues, prio)
	return result
}

func countToolUses(ctx context.Context, subStore subagentstore.Store, runID string) int {
	if subStore == nil || runID == "" {
		return 0
	}
	msgs, err := subStore.ListMessages(ctx, runID)
	if err != nil {
		return 0
	}
	n := 0
	for _, m := range msgs {
		if m.Role == role.Tool {
			n++
		}
	}
	return n
}

// HasPending reports whether any notifications are queued.
func (q *NotificationQueue) HasPending() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, v := range q.queues {
		if len(v) > 0 {
			return true
		}
	}
	return false
}
