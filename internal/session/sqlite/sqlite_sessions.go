package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hejunqiu/ds-code/internal/session"
	"time"

	"github.com/hejunqiu/ds-code/internal/llm"
)

func (s *Store) Get(ctx context.Context, id string) (session.Session, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, title, model, reasoning_effort, thinking_type, permission_mode, run_mode,
		compact_summary, compact_up_to_message_id,
		prompt_tokens_total, completion_tokens_total, prompt_cache_hit_tokens_total,
		git_snapshot, created_at, updated_at FROM sessions WHERE id=?`, id)
	return scanSession(row)
}

func scanSession(row *sql.Row) (session.Session, error) {
	var sess session.Session
	var created, updated string
	err := row.Scan(
		&sess.ID, &sess.Title, &sess.Model, &sess.ReasoningEffort, &sess.ThinkingType,
		&sess.PermissionMode, &sess.RunMode, &sess.CompactSummary, &sess.CompactUpToMessageID,
		&sess.PromptTokensTotal, &sess.CompletionTokensTotal, &sess.PromptCacheHitTokensTotal,
		&sess.GitSnapshot, &created, &updated,
	)
	if err != nil {
		return session.Session{}, err
	}
	sess.CreatedAt, err = parseRFC3339(created)
	if err != nil {
		return session.Session{}, fmt.Errorf("session created_at: %w", err)
	}
	sess.UpdatedAt, err = parseRFC3339(updated)
	if err != nil {
		return session.Session{}, fmt.Errorf("session updated_at: %w", err)
	}
	return sess, nil
}

func (s *Store) AddUsage(ctx context.Context, sessionID string, u llm.Usage) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET
		prompt_tokens_total = prompt_tokens_total + ?,
		completion_tokens_total = completion_tokens_total + ?,
		prompt_cache_hit_tokens_total = prompt_cache_hit_tokens_total + ?,
		updated_at = ?
		WHERE id=?`,
		u.PromptTokens, u.CompletionTokens, u.PromptCacheHitTokens,
		time.Now().UTC().Format(time.RFC3339), sessionID,
	)
	if err == nil {
		session.LogAddUsageDebug(sessionID, u)
	}
	return err
}

func (s *Store) UpdateSession(ctx context.Context, sessionID string, fn func(*session.Session) error) error {
	sess, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if err := fn(&sess); err != nil {
		return err
	}
	sess.UpdatedAt = time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `UPDATE sessions SET
		title=?, model=?, reasoning_effort=?, thinking_type=?, permission_mode=?, run_mode=?,
		compact_summary=?, compact_up_to_message_id=?,
		prompt_tokens_total=?, completion_tokens_total=?, prompt_cache_hit_tokens_total=?,
		git_snapshot=?, updated_at=?
		WHERE id=?`,
		sess.Title, sess.Model, sess.ReasoningEffort, sess.ThinkingType, sess.PermissionMode, sess.RunMode,
		sess.CompactSummary, sess.CompactUpToMessageID,
		sess.PromptTokensTotal, sess.CompletionTokensTotal, sess.PromptCacheHitTokensTotal,
		sess.GitSnapshot, sess.UpdatedAt.Format(time.RFC3339), sess.ID,
	)
	return err
}

func (s *Store) ListSessions(ctx context.Context, limit int) ([]session.Summary, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT s.id, s.title, s.model,
		s.prompt_tokens_total + s.completion_tokens_total,
		s.updated_at, s.created_at
		FROM sessions s
		WHERE EXISTS (SELECT 1 FROM messages m WHERE m.session_id = s.id)
		ORDER BY s.updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []session.Summary
	for rows.Next() {
		var sum session.Summary
		var updated, created string
		if err := rows.Scan(&sum.ID, &sum.Title, &sum.Model, &sum.BilledTokens, &updated, &created); err != nil {
			return nil, err
		}
		sum.UpdatedAt, err = parseRFC3339(updated)
		if err != nil {
			return nil, fmt.Errorf("session summary updated_at: %w", err)
		}
		sum.CreatedAt, err = parseRFC3339(created)
		if err != nil {
			return nil, fmt.Errorf("session summary created_at: %w", err)
		}
		if sum.Title == "" {
			sum.Title = "(untitled)"
		}
		out = append(out, sum)
	}
	return out, rows.Err()
}
