// Package subagent tracks read-only sub-agent runs (task tool) for the TUI.
package subagent

import (
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chattool"
)

// Status is the lifecycle state of a subagent run.
type Status int

const (
	StatusRunning Status = iota
	StatusDone
	StatusError
)

// Record holds UI state for one subagent exploration.
type Record struct {
	ID                 string
	Label              string
	Prompt             string
	ParentToolCallID   string
	Status             Status
	Chat      []chat.Block
	ToolLines []string
	StartedAt time.Time
	EndedAt   time.Time
	Err       string
}

// Registry stores subagent runs for the current main session turn history.
type Registry struct {
	records []*Record
}

func (r *Registry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.records)
}

func (r *Registry) All() []*Record {
	if r == nil {
		return nil
	}
	return r.records
}

func (r *Registry) Get(id string) *Record {
	if r == nil {
		return nil
	}
	for _, rec := range r.records {
		if rec.ID == id {
			return rec
		}
	}
	return nil
}

// Add appends a restored or externally built record.
func (r *Registry) Add(rec *Record) {
	if r == nil || rec == nil {
		return
	}
	r.records = append(r.records, rec)
}

func (r *Registry) Start(id, label, prompt string) *Record {
	rec := &Record{
		ID:        id,
		Label:     label,
		Prompt:    prompt,
		Status:    StatusRunning,
		StartedAt: time.Now(),
	}
	if label == "" {
		rec.Label = truncate(prompt, 48)
	}
	if prompt != "" {
		blk := chat.Block{Role: chat.RoleUser}
		blk.AppendContent(prompt)
		rec.Chat = append(rec.Chat, blk)
		rec.Chat = append(rec.Chat, chat.Block{
			Role:              chat.RoleAssistant,
			Streaming:         true,
			ReasoningOpen:     false,
			ReasoningStartedAt: time.Now(),
		})
	}
	if r == nil {
		return rec
	}
	r.records = append(r.records, rec)
	return rec
}

func (r *Registry) End(id, summary string, runErr error) {
	rec := r.Get(id)
	if rec == nil {
		return
	}
	now := time.Now()
	rec.EndedAt = now
	finalizeAssistant(rec, now)
	for i := range rec.Chat {
		if rec.Chat[i].Role == chat.RoleTool && rec.Chat[i].ToolRunning {
			rec.Chat[i].ToolRunning = false
		}
	}
	if runErr != nil {
		rec.Status = StatusError
		rec.Err = runErr.Error()
		blk := chat.Block{Role: chat.RoleTool, ToolName: "task", ToolResult: rec.Err, ToolError: true}
		rec.Chat = append(rec.Chat, blk)
		return
	}
	rec.Status = StatusDone
	if summary != "" {
		blk := chat.Block{Role: chat.RoleAssistant}
		blk.AppendContent(summary)
		rec.Chat = append(rec.Chat, blk)
	}
}

func (r *Registry) ToolStart(id, name, args, command string) {
	rec := r.Get(id)
	if rec == nil {
		return
	}
	appendToolBlock(rec, name, args, command, "", true, false)
	rec.ToolLines = append(rec.ToolLines, chattool.Line(name, args, command, "", true, false))
}

func (r *Registry) ToolEnd(id, name, args, command, result string, isError bool) {
	rec := r.Get(id)
	if rec == nil {
		return
	}
	finishToolBlock(rec, name, args, command, result, isError)
	rec.ToolLines = rec.ToolLines[:0]
	for i := range rec.Chat {
		b := &rec.Chat[i]
		if b.Role == chat.RoleTool {
			preview := b.ToolResult
			if preview == "" && b.ToolRunning {
				preview = "…"
			}
			rec.ToolLines = append(rec.ToolLines, chattool.Line(b.ToolName, b.ToolArgs, b.ToolCommand, preview, b.ToolRunning, b.ToolError))
		}
	}
}

func appendToolBlock(rec *Record, name, args, command, result string, running, isError bool) {
	if len(rec.Chat) > 0 {
		last := &rec.Chat[len(rec.Chat)-1]
		if last.Role == chat.RoleAssistant {
			last.FinalizeReasoning(time.Now())
			last.Streaming = false
		}
	}
	rec.Chat = append(rec.Chat, chat.Block{
		Role:        chat.RoleTool,
		ToolName:    name,
		ToolArgs:    args,
		ToolCommand: command,
		ToolResult:  result,
		ToolRunning: running,
		ToolError:   isError,
	})
}

func finishToolBlock(rec *Record, name, args, command, result string, isError bool) {
	for i := len(rec.Chat) - 1; i >= 0; i-- {
		if rec.Chat[i].Role != chat.RoleTool || !rec.Chat[i].ToolRunning {
			continue
		}
		rec.Chat[i].ToolName = name
		rec.Chat[i].ToolArgs = args
		rec.Chat[i].ToolCommand = command
		rec.Chat[i].ToolResult = result
		rec.Chat[i].ToolRunning = false
		rec.Chat[i].ToolError = isError
		return
	}
	appendToolBlock(rec, name, args, command, result, false, isError)
}

func finalizeAssistant(rec *Record, at time.Time) {
	for i := len(rec.Chat) - 1; i >= 0; i-- {
		if rec.Chat[i].Role != chat.RoleAssistant {
			continue
		}
		rec.Chat[i].FinalizeReasoning(at)
		rec.Chat[i].Streaming = false
		return
	}
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
