package agent

import (
	"context"
	"encoding/json"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/readgate"
	"github.com/wzhejunqiu/ds-code/internal/toolresult"
)

func (r *Runner) ensureReadPaths(sessionID string) map[string]struct{} {
	r.readPathsMu.Lock()
	defer r.readPathsMu.Unlock()
	if r.readPaths == nil {
		r.readPaths = make(map[string]map[string]struct{})
	}
	if r.readPathsHydrated == nil {
		r.readPathsHydrated = make(map[string]bool)
	}
	set, ok := r.readPaths[sessionID]
	if !ok {
		set = make(map[string]struct{})
		r.readPaths[sessionID] = set
	}
	if !r.readPathsHydrated[sessionID] {
		r.readPathsHydrated[sessionID] = true
		if r.Sessions != nil && r.Perm != nil {
			msgs, err := r.Sessions.ListMessages(context.Background(), sessionID)
			if err == nil {
				for p := range HydrateReadPaths(r.Perm.Workspace, msgs) {
					set[p] = struct{}{}
				}
			}
		}
	}
	return set
}

func (r *Runner) readPathSnapshot(sessionID string) map[string]struct{} {
	live := r.ensureReadPaths(sessionID)
	out := make(map[string]struct{}, len(live))
	for p := range live {
		out[p] = struct{}{}
	}
	return out
}

func (r *Runner) markReadPath(sessionID, canonical string) {
	if canonical == "" {
		return
	}
	r.readPathsMu.Lock()
	defer r.readPathsMu.Unlock()
	if r.readPaths == nil {
		r.readPaths = make(map[string]map[string]struct{})
	}
	set, ok := r.readPaths[sessionID]
	if !ok {
		set = make(map[string]struct{})
		r.readPaths[sessionID] = set
	}
	set[canonical] = struct{}{}
}

func (r *Runner) readGateForSubRound(ctx context.Context, sessionID string, toolCalls []llm.ToolCall) context.Context {
	if r.Perm == nil {
		return ctx
	}
	snapshot := r.readPathSnapshot(sessionID)
	sameBatch := collectSameBatchReadPaths(r.Perm.Workspace, toolCalls)
	gate := readgate.NewGate(r.Perm.Workspace, snapshot, sameBatch, func(canon string) {
		r.markReadPath(sessionID, canon)
	})
	return readgate.WithGate(ctx, gate)
}

func collectSameBatchReadPaths(workspace string, toolCalls []llm.ToolCall) map[string]struct{} {
	out := make(map[string]struct{})
	for _, tc := range toolCalls {
		if !tool.NameReadFile.Matches(tc.Name) {
			continue
		}
		var in struct {
			Filepath string `json:"filepath"`
		}
		if err := json.Unmarshal([]byte(tc.Arguments), &in); err != nil || in.Filepath == "" {
			continue
		}
		if canon, err := readgate.CanonicalPath(workspace, in.Filepath); err == nil {
			out[canon] = struct{}{}
		}
	}
	return out
}

// HydrateReadPaths rebuilds the read set from persisted session messages.
func HydrateReadPaths(workspace string, msgs []session.Message) map[string]struct{} {
	out := make(map[string]struct{})
	if workspace == "" {
		return out
	}
	readArgs := make(map[string]string) // tool_call_id -> filepath
	for _, m := range msgs {
		if m.Role == role.Assistant && m.ToolCallsJSON != "" {
			var calls []llm.ToolCall
			if err := json.Unmarshal([]byte(m.ToolCallsJSON), &calls); err != nil {
				continue
			}
			for _, tc := range calls {
				if !tool.NameReadFile.Matches(tc.Name) {
					continue
				}
				var in struct {
					Filepath string `json:"filepath"`
				}
				if err := json.Unmarshal([]byte(tc.Arguments), &in); err != nil || in.Filepath == "" {
					continue
				}
				readArgs[tc.ID] = in.Filepath
			}
		}
	}
	for _, m := range msgs {
		if m.Role != role.Tool || !tool.NameReadFile.Matches(m.ToolName) {
			continue
		}
		_, isErr := toolresult.UnpackToolBody(m.Content)
		if isErr {
			continue
		}
		fp, ok := readArgs[m.ToolCallID]
		if !ok || fp == "" {
			continue
		}
		if canon, err := readgate.CanonicalPath(workspace, fp); err == nil {
			out[canon] = struct{}{}
		}
	}
	return out
}
