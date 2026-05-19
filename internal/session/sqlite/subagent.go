package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session/subagentstore"
)

// SubagentStore returns a subagent store backed by the same database.
func (s *Store) SubagentStore() subagentstore.Store {
	return &subagentSQLite{db: s.db}
}

type subagentSQLite struct {
	db *sql.DB
}

func (st *subagentSQLite) CreateRun(_ context.Context, p subagentstore.CreateRunParams) (subagentstore.Run, error) {
	now := time.Now().UTC()
	id := "sa-" + uuid.NewString()
	_, err := st.db.Exec(`INSERT INTO subagent_runs (
		id, parent_session_id, parent_tool_call_id, label, prompt, status, error,
		model, reasoning_effort, thinking_type,
		prompt_tokens_total, completion_tokens_total, prompt_cache_hit_tokens_total,
		created_at, ended_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, p.ParentSessionID, p.ParentToolCallID, p.Label, p.Prompt, string(subagentstore.StatusRunning), "",
		p.Model, p.ReasoningEffort, p.ThinkingType,
		0, 0, 0,
		now.Format(time.RFC3339), "",
	)
	if err != nil {
		return subagentstore.Run{}, err
	}
	return subagentstore.Run{
		ID:               id,
		ParentSessionID:  p.ParentSessionID,
		ParentToolCallID: p.ParentToolCallID,
		Label:            p.Label,
		Prompt:           p.Prompt,
		Status:           subagentstore.StatusRunning,
		Model:            p.Model,
		ReasoningEffort:  p.ReasoningEffort,
		ThinkingType:     p.ThinkingType,
		CreatedAt:        now,
	}, nil
}

func (st *subagentSQLite) FinishRun(_ context.Context, runID string, status subagentstore.Status, errMsg string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := st.db.Exec(`UPDATE subagent_runs SET status=?, error=?, ended_at=? WHERE id=?`,
		string(status), errMsg, now, runID,
	)
	return err
}

func (st *subagentSQLite) ListRuns(_ context.Context, parentSessionID string) ([]subagentstore.Run, error) {
	rows, err := st.db.Query(`SELECT id, parent_session_id, parent_tool_call_id, label, prompt, status, error,
		model, reasoning_effort, thinking_type,
		prompt_tokens_total, completion_tokens_total, prompt_cache_hit_tokens_total,
		created_at, ended_at
		FROM subagent_runs WHERE parent_session_id=? ORDER BY created_at ASC`, parentSessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRuns(rows)
}

func (st *subagentSQLite) GetRun(_ context.Context, runID string) (subagentstore.Run, error) {
	row := st.db.QueryRow(`SELECT id, parent_session_id, parent_tool_call_id, label, prompt, status, error,
		model, reasoning_effort, thinking_type,
		prompt_tokens_total, completion_tokens_total, prompt_cache_hit_tokens_total,
		created_at, ended_at
		FROM subagent_runs WHERE id=?`, runID)
	return scanRun(row)
}

func (st *subagentSQLite) GetRunByToolCall(_ context.Context, parentSessionID, parentToolCallID string) (subagentstore.Run, error) {
	row := st.db.QueryRow(`SELECT id, parent_session_id, parent_tool_call_id, label, prompt, status, error,
		model, reasoning_effort, thinking_type,
		prompt_tokens_total, completion_tokens_total, prompt_cache_hit_tokens_total,
		created_at, ended_at
		FROM subagent_runs WHERE parent_session_id=? AND parent_tool_call_id=?`,
		parentSessionID, parentToolCallID,
	)
	return scanRun(row)
}

func (st *subagentSQLite) AppendMessage(_ context.Context, msg subagentstore.Message) error {
	now := time.Now().UTC()
	if !msg.CreatedAt.IsZero() {
		now = msg.CreatedAt
	}
	_, err := st.db.Exec(`INSERT INTO subagent_messages (
		run_id, role, content, reasoning_content, reasoning_duration_ms, turn_duration_ms,
		tool_calls_json, tool_call_id, tool_name,
		prompt_tokens, completion_tokens, prompt_cache_hit_tokens, created_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		msg.RunID, msg.Role, msg.Content, msg.ReasoningContent, msg.ReasoningDurationMS, msg.TurnDurationMS,
		msg.ToolCallsJSON, msg.ToolCallID, msg.ToolName,
		msg.PromptTokens, msg.CompletionTokens, msg.PromptCacheHitTokens,
		now.Format(time.RFC3339),
	)
	return err
}

func (st *subagentSQLite) ListMessages(_ context.Context, runID string) ([]subagentstore.Message, error) {
	rows, err := st.db.Query(`SELECT id, run_id, role, content,
		COALESCE(reasoning_content, ''),
		COALESCE(reasoning_duration_ms, 0), COALESCE(turn_duration_ms, 0),
		COALESCE(tool_calls_json, ''),
		COALESCE(tool_call_id, ''), COALESCE(tool_name, ''),
		prompt_tokens, completion_tokens, prompt_cache_hit_tokens, created_at
		FROM subagent_messages WHERE run_id=? ORDER BY id ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []subagentstore.Message
	for rows.Next() {
		m, err := scanSubagentMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (st *subagentSQLite) AddUsage(_ context.Context, runID string, u llm.Usage) error {
	_, err := st.db.Exec(`UPDATE subagent_runs SET
		prompt_tokens_total = prompt_tokens_total + ?,
		completion_tokens_total = completion_tokens_total + ?,
		prompt_cache_hit_tokens_total = prompt_cache_hit_tokens_total + ?
		WHERE id=?`,
		u.PromptTokens, u.CompletionTokens, u.PromptCacheHitTokens, runID,
	)
	return err
}

func (st *subagentSQLite) SumUsage(_ context.Context, parentSessionID string) (llm.Usage, error) {
	row := st.db.QueryRow(`SELECT
		COALESCE(SUM(prompt_tokens_total), 0),
		COALESCE(SUM(completion_tokens_total), 0),
		COALESCE(SUM(prompt_cache_hit_tokens_total), 0)
		FROM subagent_runs WHERE parent_session_id=?`, parentSessionID)
	var u llm.Usage
	var p, c, cache int64
	if err := row.Scan(&p, &c, &cache); err != nil {
		return u, err
	}
	u.PromptTokens = int(p)
	u.CompletionTokens = int(c)
	u.PromptCacheHitTokens = int(cache)
	return u, nil
}

func scanRuns(rows *sql.Rows) ([]subagentstore.Run, error) {
	var out []subagentstore.Run
	for rows.Next() {
		r, err := scanRunRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanRun(row *sql.Row) (subagentstore.Run, error) {
	return scanRunRow(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRunRow(row rowScanner) (subagentstore.Run, error) {
	var r subagentstore.Run
	var status, created, ended string
	err := row.Scan(
		&r.ID, &r.ParentSessionID, &r.ParentToolCallID, &r.Label, &r.Prompt, &status, &r.Error,
		&r.Model, &r.ReasoningEffort, &r.ThinkingType,
		&r.PromptTokensTotal, &r.CompletionTokensTotal, &r.PromptCacheHitTokensTotal,
		&created, &ended,
	)
	if err != nil {
		return subagentstore.Run{}, err
	}
	r.Status = subagentstore.Status(status)
	r.CreatedAt, err = parseRFC3339(created)
	if err != nil {
		return subagentstore.Run{}, err
	}
	if ended != "" {
		r.EndedAt, err = parseRFC3339(ended)
		if err != nil {
			return subagentstore.Run{}, err
		}
	}
	return r, nil
}

func scanSubagentMessage(rows *sql.Rows) (subagentstore.Message, error) {
	var m subagentstore.Message
	var roleStr, created string
	err := rows.Scan(
		&m.ID, &m.RunID, &roleStr, &m.Content,
		&m.ReasoningContent, &m.ReasoningDurationMS, &m.TurnDurationMS,
		&m.ToolCallsJSON, &m.ToolCallID, &m.ToolName,
		&m.PromptTokens, &m.CompletionTokens, &m.PromptCacheHitTokens, &created,
	)
	if err != nil {
		return subagentstore.Message{}, err
	}
	m.Role = role.Role(roleStr)
	m.CreatedAt, err = parseRFC3339(created)
	return m, err
}
