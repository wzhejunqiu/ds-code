// Package bridge adapts agent.TurnCallbacks to Wails Events using Envelope v1.
package bridge

import "encoding/json"

const (
	EventTopic      = "agent:event"
	EnvelopeVersion = 1
)

// AgentEventKind identifies the event payload shape.
type AgentEventKind string

const (
	KindTurnStarted         AgentEventKind = "turn.started"
	KindContentDelta        AgentEventKind = "content.delta"
	KindReasoningDelta      AgentEventKind = "reasoning.delta"
	KindToolStart           AgentEventKind = "tool.start"
	KindToolEnd             AgentEventKind = "tool.end"
	KindAssistantSegmentEnd AgentEventKind = "assistant.segment_end"
	KindPlanningStart       AgentEventKind = "planning.start"
	KindPlanningEnd         AgentEventKind = "planning.end"
	KindSubagentStart       AgentEventKind = "subagent.start"
	KindSubagentEnd         AgentEventKind = "subagent.end"
	KindSubagentToolStart   AgentEventKind = "subagent.tool.start"
	KindSubagentToolEnd     AgentEventKind = "subagent.tool.end"
	KindUsageUpdate         AgentEventKind = "usage.update"
	KindTurnDone            AgentEventKind = "turn.done"
	KindPermissionRequest   AgentEventKind = "permission.request"
	KindSystemNotice        AgentEventKind = "system.notice"
)

// AgentEventEnvelope is the versioned event wrapper sent over Wails Events.
type AgentEventEnvelope struct {
	V           int             `json:"v"`
	Seq         uint64          `json:"seq"`
	TurnID      string          `json:"turnId"`
	StreamID    string          `json:"streamId"`
	WorkspaceID string          `json:"workspaceId"`
	Kind        AgentEventKind  `json:"kind"`
	Ts          int64           `json:"ts"`
	Critical    bool            `json:"critical"`
	Payload     json.RawMessage `json:"payload"`
}

type ContentDeltaPayload struct {
	Delta string `json:"delta"`
}

type ReasoningDeltaPayload struct {
	Delta string `json:"delta"`
}

type ToolStartPayload struct {
	Name            string `json:"name"`
	Args            string `json:"args"`
	Command         string `json:"command,omitempty"`
	TimeoutDeadline int64  `json:"timeoutDeadline,omitempty"`
}

type ToolEndPayload struct {
	Name    string `json:"name"`
	Args    string `json:"args"`
	Command string `json:"command,omitempty"`
	Result  string `json:"result"`
	IsError bool   `json:"isError"`
}

type TurnStartedPayload struct {
	SessionID string `json:"sessionId"`
}

type TurnDonePayload struct {
	Error      string `json:"error,omitempty"`
	Cancelled  bool   `json:"cancelled,omitempty"`
	SubRounds  int    `json:"subRounds,omitempty"`
	FinalChars int    `json:"finalChars,omitempty"`
}

type PermissionRequestPayload struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Tool    string `json:"tool"`
	Summary string `json:"summary"`
	Host    string `json:"host,omitempty"`
	URL     string `json:"url,omitempty"`
}

type SubagentStartPayload struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Prompt     string `json:"prompt"`
	AgentType  string `json:"agentType"`
	Background bool   `json:"background"`
}

type SubagentEndPayload struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
	Error   string `json:"error,omitempty"`
}

type SubagentToolStartPayload struct {
	SubagentID string `json:"subagentId"`
	Name       string `json:"name"`
	Args       string `json:"args"`
	Command    string `json:"command,omitempty"`
}

type SystemNoticePayload struct {
	Text string `json:"text"`
}

type SubagentToolEndPayload struct {
	SubagentID string `json:"subagentId"`
	Name       string `json:"name"`
	Args       string `json:"args"`
	Command    string `json:"command,omitempty"`
	Result     string `json:"result"`
	IsError    bool   `json:"isError"`
}

func MustPayload(v any) json.RawMessage {
	return mustPayload(v)
}

func mustPayload(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}
