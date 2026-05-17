package session

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

const schemaVersion = 2

// SQLiteStore persists sessions in SQLite (Phase 3+).
type SQLiteStore struct {
	db *sql.DB
}

// OpenSQLite opens or creates sessions.db at path (0600).
func OpenSQLite(path string) (*SQLiteStore, error) {
	if err := os.MkdirAll(dirOf(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	s := &SQLiteStore{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func dirOf(path string) string {
	if i := strings.LastIndex(path, string(os.PathSeparator)); i >= 0 {
		return path[:i]
	}
	return "."
}

func (s *SQLiteStore) initSchema() error {
	var v int
	err := s.db.QueryRow(`SELECT version FROM schema_version LIMIT 1`).Scan(&v)
	switch {
	case err == sql.ErrNoRows:
		return s.applySchema()
	case err != nil:
		// Fresh or legacy file without schema_version — initialize current schema.
		return s.applySchema()
	case v != schemaVersion:
		return fmt.Errorf(
			"sessions.db schema version %d (expected %d): delete the database file and restart",
			v, schemaVersion,
		)
	default:
		return nil
	}
}

func (s *SQLiteStore) applySchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`,
		`DELETE FROM schema_version`,
		`INSERT INTO schema_version (version) VALUES (` + fmt.Sprint(schemaVersion) + `)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL,
			reasoning_effort TEXT NOT NULL,
			thinking_type TEXT NOT NULL,
			permission_mode TEXT NOT NULL,
			run_mode TEXT NOT NULL,
			compact_summary TEXT NOT NULL DEFAULT '',
			compact_up_to_message_id INTEGER NOT NULL DEFAULT 0,
			prompt_tokens_total INTEGER NOT NULL DEFAULT 0,
			completion_tokens_total INTEGER NOT NULL DEFAULT 0,
			prompt_cache_hit_tokens_total INTEGER NOT NULL DEFAULT 0,
			git_snapshot TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			reasoning_content TEXT NOT NULL DEFAULT '',
			reasoning_duration_ms INTEGER NOT NULL DEFAULT 0,
			turn_duration_ms INTEGER NOT NULL DEFAULT 0,
			tool_calls_json TEXT NOT NULL DEFAULT '',
			tool_call_id TEXT NOT NULL DEFAULT '',
			tool_name TEXT NOT NULL DEFAULT '',
			prompt_tokens INTEGER NOT NULL DEFAULT 0,
			completion_tokens INTEGER NOT NULL DEFAULT 0,
			prompt_cache_hit_tokens INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id, id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(session_id, created_at)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) CreateSession(model, effort, thinking, permMode, runMode string) (Session, error) {
	now := time.Now().UTC()
	sess := Session{
		ID:              uuid.NewString(),
		Model:           model,
		ReasoningEffort: effort,
		ThinkingType:    thinking,
		PermissionMode:  permMode,
		RunMode:         runMode,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.Create(context.Background(), sess); err != nil {
		return Session{}, err
	}
	return sess, nil
}

func (s *SQLiteStore) Create(_ context.Context, sess Session) error {
	_, err := s.db.Exec(`INSERT INTO sessions (
		id, title, model, reasoning_effort, thinking_type, permission_mode, run_mode,
		compact_summary, compact_up_to_message_id,
		prompt_tokens_total, completion_tokens_total, prompt_cache_hit_tokens_total,
		git_snapshot, created_at, updated_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		sess.ID, sess.Title, sess.Model, sess.ReasoningEffort, sess.ThinkingType,
		sess.PermissionMode, sess.RunMode, sess.CompactSummary, sess.CompactUpToMessageID,
		sess.PromptTokensTotal, sess.CompletionTokensTotal, sess.PromptCacheHitTokensTotal,
		sess.GitSnapshot, sess.CreatedAt.Format(time.RFC3339), sess.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) Get(_ context.Context, id string) (Session, error) {
	row := s.db.QueryRow(`SELECT id, title, model, reasoning_effort, thinking_type, permission_mode, run_mode,
		compact_summary, compact_up_to_message_id,
		prompt_tokens_total, completion_tokens_total, prompt_cache_hit_tokens_total,
		git_snapshot, created_at, updated_at FROM sessions WHERE id=?`, id)
	return scanSession(row)
}

func scanSession(row *sql.Row) (Session, error) {
	var sess Session
	var created, updated string
	err := row.Scan(
		&sess.ID, &sess.Title, &sess.Model, &sess.ReasoningEffort, &sess.ThinkingType,
		&sess.PermissionMode, &sess.RunMode, &sess.CompactSummary, &sess.CompactUpToMessageID,
		&sess.PromptTokensTotal, &sess.CompletionTokensTotal, &sess.PromptCacheHitTokensTotal,
		&sess.GitSnapshot, &created, &updated,
	)
	if err != nil {
		return Session{}, err
	}
	sess.CreatedAt, _ = time.Parse(time.RFC3339, created)
	sess.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return sess, nil
}

func (s *SQLiteStore) ListMessages(_ context.Context, sessionID string) ([]Message, error) {
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
	var out []Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func scanMessage(rows *sql.Rows) (Message, error) {
	var m Message
	var created string
	err := rows.Scan(
		&m.ID, &m.SessionID, &m.Role, &m.Content, &m.ReasoningContent,
		&m.ReasoningDurationMS, &m.TurnDurationMS,
		&m.ToolCallsJSON, &m.ToolCallID, &m.ToolName,
		&m.PromptTokens, &m.CompletionTokens, &m.PromptCacheHitTokens, &created,
	)
	if err != nil {
		return Message{}, err
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return m, nil
}

func (s *SQLiteStore) AppendMessage(_ context.Context, msg Message) error {
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

	if msg.Role == "user" {
		title := truncateTitle(msg.Content, 80)
		_, _ = s.db.Exec(`UPDATE sessions SET title=CASE WHEN title='' THEN ? ELSE title END, updated_at=? WHERE id=?`,
			title, now.Format(time.RFC3339), msg.SessionID)
	} else {
		_, _ = s.db.Exec(`UPDATE sessions SET updated_at=? WHERE id=?`, now.Format(time.RFC3339), msg.SessionID)
	}
	return nil
}

func truncateTitle(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (s *SQLiteStore) AddUsage(_ context.Context, sessionID string, u llm.Usage) error {
	_, err := s.db.Exec(`UPDATE sessions SET
		prompt_tokens_total = prompt_tokens_total + ?,
		completion_tokens_total = completion_tokens_total + ?,
		prompt_cache_hit_tokens_total = prompt_cache_hit_tokens_total + ?,
		updated_at = ?
		WHERE id=?`,
		u.PromptTokens, u.CompletionTokens, u.PromptCacheHitTokens,
		time.Now().UTC().Format(time.RFC3339), sessionID,
	)
	return err
}

func (s *SQLiteStore) UpdateSession(_ context.Context, sessionID string, fn func(*Session) error) error {
	sess, err := s.Get(context.Background(), sessionID)
	if err != nil {
		return err
	}
	if err := fn(&sess); err != nil {
		return err
	}
	sess.UpdatedAt = time.Now().UTC()
	_, err = s.db.Exec(`UPDATE sessions SET
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

func (s *SQLiteStore) ListSessions(_ context.Context, limit int) ([]Summary, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT s.id, s.title, s.model,
		s.prompt_tokens_total + s.completion_tokens_total,
		s.updated_at, s.created_at
		FROM sessions s
		WHERE EXISTS (SELECT 1 FROM messages m WHERE m.session_id = s.id)
		ORDER BY s.updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Summary
	for rows.Next() {
		var sum Summary
		var updated, created string
		if err := rows.Scan(&sum.ID, &sum.Title, &sum.Model, &sum.BilledTokens, &updated, &created); err != nil {
			return nil, err
		}
		sum.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		sum.CreatedAt, _ = time.Parse(time.RFC3339, created)
		if sum.Title == "" {
			sum.Title = "(untitled)"
		}
		out = append(out, sum)
	}
	return out, rows.Err()
}
