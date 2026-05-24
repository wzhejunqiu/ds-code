package spawn

import (
	"fmt"
	"sync"

	"github.com/wzhejunqiu/ds-code/internal/llm"
)

// NotificationPriority controls when a notification is drained into the main conversation.
type NotificationPriority int

const (
	PrioNow   NotificationPriority = iota // injected before next user message
	PrioNext                               // drained at start of next RunTurn
	PrioLater                              // drained when idle
)

// Notification is a completion/failure/kill notice for an async agent.
type Notification struct {
	AgentID     string
	ToolUseID   string
	OutputFile  string
	Status      string // completed | failed | killed
	Summary     string
	Result      string
	Usage       llm.Usage
	DurationMS  int64
	ToolUseCount int
	WorktreePath   string
	WorktreeBranch string
}

// FormatXML renders the notification as an XML task-notification block.
func (n Notification) FormatXML() string {
	wt := ""
	if n.WorktreePath != "" {
		wt = fmt.Sprintf("\n  <worktree><worktreePath>%s</worktreePath><worktreeBranch>%s</worktreeBranch></worktree>",
			n.WorktreePath, n.WorktreeBranch)
	}
	return fmt.Sprintf(
		`<task-notification>
  <task-id>%s</task-id>
  <tool-use-id>%s</tool-use-id>
  <output-file>%s</output-file>
  <status>%s</status>
  <summary>%s</summary>
  <result>%s</result>
  <usage><total_tokens>%d</total_tokens><tool_uses>%d</tool_uses><duration_ms>%d</duration_ms></usage>%s
</task-notification>`,
		n.AgentID, n.ToolUseID, n.OutputFile, n.Status, n.Summary, n.Result,
		n.Usage.PromptTokens+n.Usage.CompletionTokens, n.ToolUseCount, n.DurationMS,
		wt,
	)
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
