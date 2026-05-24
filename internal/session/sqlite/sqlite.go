package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/session"
	_ "modernc.org/sqlite"
)

// Store persists sessions in SQLite (Phase 3+).
type Store struct {
	db   *sql.DB
	path string
}

// Open opens or creates sessions.db at path (0600).
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(dirOf(path), 0o700); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.WriteFile(path, nil, 0o600); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else if err := os.Chmod(path, 0o600); err != nil {
		return nil, fmt.Errorf("chmod sessions.db: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, path: path}
	db.SetMaxOpenConns(1)
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil && !os.IsNotExist(err) {
		db.Close()
		return nil, fmt.Errorf("chmod sessions.db: %w", err)
	}
	return s, nil
}

func dirOf(path string) string {
	if i := strings.LastIndex(path, string(os.PathSeparator)); i >= 0 {
		return path[:i]
	}
	return "."
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) CreateSession(model, effort, thinking, permMode, runMode string) (session.Session, error) {
	now := time.Now().UTC()
	snap := billing.SnapshotForModel(model)
	sess := session.Session{
		ID:                  uuid.NewString(),
		Model:               model,
		ReasoningEffort:     effort,
		ThinkingType:        thinking,
		PermissionMode:      permMode,
		RunMode:             runMode,
		PricingSnapshotJSON: billing.MarshalSnapshot(snap),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := s.Create(context.Background(), sess); err != nil {
		return session.Session{}, err
	}
	return sess, nil
}

func (s *Store) Create(ctx context.Context, sess session.Session) error {
	if sess.PricingSnapshotJSON == "" && sess.Model != "" {
		snap := billing.SnapshotForModel(sess.Model)
		sess.PricingSnapshotJSON = billing.MarshalSnapshot(snap)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions (
		id, title, model, reasoning_effort, thinking_type, permission_mode, run_mode,
		compact_summary, compact_up_to_message_id,
		prompt_tokens_total, completion_tokens_total, prompt_cache_hit_tokens_total,
		pricing_snapshot_json, estimated_cost_cny,
		git_snapshot, created_at, updated_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		sess.ID, sess.Title, sess.Model, sess.ReasoningEffort, sess.ThinkingType,
		sess.PermissionMode, sess.RunMode, sess.CompactSummary, sess.CompactUpToMessageID,
		sess.PromptTokensTotal, sess.CompletionTokensTotal, sess.PromptCacheHitTokensTotal,
		sess.PricingSnapshotJSON, sess.EstimatedCostCNY,
		sess.GitSnapshot, sess.CreatedAt.Format(time.RFC3339), sess.UpdatedAt.Format(time.RFC3339),
	)
	return err
}
