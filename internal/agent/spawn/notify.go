package spawn

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
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
	AgentID        string
	ToolUseID      string
	OutputFile     string
	Status         string // completed | failed | killed
	Summary        string
	Result         string
	Usage          llm.Usage
	DurationMS     int64
	ToolUseCount   int
	WorktreePath   string
	WorktreeBranch string
}

type notificationPayload struct {
	AgentID    string                    `json:"agent_id"`
	ToolUseID  string                    `json:"tool_use_id"`
	OutputFile string                    `json:"output_file,omitempty"`
	Status     string                    `json:"status"`
	Summary    string                    `json:"summary"`
	Result     string                    `json:"result,omitempty"`
	Usage      notificationUsagePayload  `json:"usage"`
	Worktree   *notificationWorktree     `json:"worktree,omitempty"`
}

type notificationUsagePayload struct {
	TotalTokens int   `json:"total_tokens"`
	ToolUses    int   `json:"tool_uses"`
	DurationMS  int64 `json:"duration_ms"`
}

type notificationWorktree struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
}

// notificationPriority picks PrioLater during an active parent turn, else PrioNext.
func notificationPriority(ctx context.Context) NotificationPriority {
	if agent.InActiveTurn(ctx) {
		return PrioLater
	}
	return PrioNext
}

// Format renders the notification as a tagged JSON block for LLM consumption.
func (n Notification) Format() string {
	payload := notificationPayload{
		AgentID:    n.AgentID,
		ToolUseID:  n.ToolUseID,
		OutputFile: n.OutputFile,
		Status:     n.Status,
		Summary:    n.Summary,
		Result:     n.Result,
		Usage: notificationUsagePayload{
			TotalTokens: n.Usage.PromptTokens + n.Usage.CompletionTokens,
			ToolUses:    n.ToolUseCount,
			DurationMS:  n.DurationMS,
		},
	}
	if n.WorktreePath != "" {
		payload.Worktree = &notificationWorktree{
			Path:   n.WorktreePath,
			Branch: n.WorktreeBranch,
		}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		b = []byte(fmt.Sprintf(`{"agent_id":%q,"status":%q,"summary":"serialization error"}`, n.AgentID, n.Status))
	}
	return fmt.Sprintf("<task-notification>\n%s\n</task-notification>", b)
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
