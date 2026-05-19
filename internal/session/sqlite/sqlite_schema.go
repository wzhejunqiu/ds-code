package sqlite

import (
	"database/sql"
	"fmt"
)

const schemaVersion = 3

func (s *Store) initSchema() error {
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

func (s *Store) applySchema() error {
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
		`CREATE TABLE IF NOT EXISTS subagent_runs (
			id TEXT PRIMARY KEY,
			parent_session_id TEXT NOT NULL,
			parent_tool_call_id TEXT NOT NULL,
			label TEXT NOT NULL DEFAULT '',
			prompt TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'running',
			error TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			reasoning_effort TEXT NOT NULL DEFAULT '',
			thinking_type TEXT NOT NULL DEFAULT '',
			prompt_tokens_total INTEGER NOT NULL DEFAULT 0,
			completion_tokens_total INTEGER NOT NULL DEFAULT 0,
			prompt_cache_hit_tokens_total INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			ended_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_subagent_runs_parent ON subagent_runs(parent_session_id, created_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_subagent_runs_tool_call ON subagent_runs(parent_session_id, parent_tool_call_id)`,
		`CREATE TABLE IF NOT EXISTS subagent_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL,
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
			FOREIGN KEY (run_id) REFERENCES subagent_runs(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_subagent_messages_run ON subagent_messages(run_id, id)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}
