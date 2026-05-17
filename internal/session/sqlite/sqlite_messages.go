package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hejunqiu/ds-code/internal/session"
	"time"

	"github.com/hejunqiu/ds-code/internal/role"
)

func (s *Store) ListMessages(_ context.Context, sessionID string) ([]session.Message, error) {
	rows, err := s.db.Query(`SELECT id, session_id, role, content,
		COALESCE(reasoning_content, ''),
		COALESCE(reasoning_duration_ms, 0), COALESCE(turn_duration_ms, 0),
		COALESCE(tool_calls_json, ''),
		COALESCE(tool_call_id, ''), COALESCE(tool_name, ''),
		prompt_tokens, completion_tokens, prompt_cache_hit_tokens, created_at
		FROM messages WHERE session_id=? ORDER BY id ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []session.Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func scanMessage(rows *sql.Rows) (session.Message, error) {
	var m session.Message
	var created string
	err := rows.Scan(
		&m.ID, &m.SessionID, &m.Role, &m.Content, &m.ReasoningContent,
		&m.ReasoningDurationMS, &m.TurnDurationMS,
		&m.ToolCallsJSON, &m.ToolCallID, &m.ToolName,
		&m.PromptTokens, &m.CompletionTokens, &m.PromptCacheHitTokens, &created,
	)
	if err != nil {
		return session.Message{}, err
	}
	m.CreatedAt, err = parseRFC3339(created)
	if err != nil {
		return session.Message{}, fmt.Errorf("message created_at: %w", err)
	}
	return m, nil
}

func (s *Store) AppendMessage(_ context.Context, msg session.Message) error {
	now := time.Now().UTC()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	res, err := s.db.Exec(`INSERT INTO messages (
		session_id, role, content, reasoning_content, reasoning_duration_ms, turn_duration_ms,
		tool_calls_json, tool_call_id, tool_name,
		prompt_tokens, completion_tokens, prompt_cache_hit_tokens, created_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		msg.SessionID, msg.Role, msg.Content, msg.ReasoningContent,
		msg.ReasoningDurationMS, msg.TurnDurationMS,
		msg.ToolCallsJSON, msg.ToolCallID, msg.ToolName,
		msg.PromptTokens, msg.CompletionTokens, msg.PromptCacheHitTokens,
		msg.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	msg.ID = id

	if msg.Role == role.User {
		title := session.TruncateTitle(msg.Content, 80)
		_, _ = s.db.Exec(`UPDATE sessions SET title=CASE WHEN title='' THEN ? ELSE title END, updated_at=? WHERE id=?`,
			title, now.Format(time.RFC3339), msg.SessionID)
	} else {
		_, _ = s.db.Exec(`UPDATE sessions SET updated_at=? WHERE id=?`, now.Format(time.RFC3339), msg.SessionID)
	}
	return nil
}
