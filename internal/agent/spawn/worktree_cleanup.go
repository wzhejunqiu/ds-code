package spawn

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"go.uber.org/zap"
)

func (s *Service) worktreeTTL() time.Duration {
	ttl := s.Cfg.Tools.Agent.WorktreeTTL
	if ttl <= 0 {
		return 24 * time.Hour
	}
	return ttl
}

// cleanupWorktreeImmediate removes a worktree without TTL delay (errors, rollback).
func (s *Service) cleanupWorktreeImmediate(ctx context.Context, run subagentstore.Run) {
	if s.Worktrees == nil || run.WorktreePath == "" {
		return
	}
	if err := s.Worktrees.ValidatePath(run.WorktreePath); err != nil {
		logging.L().Warn("skip worktree remove: path outside base dir",
			zap.String("path", run.WorktreePath),
			zap.Error(err),
		)
		return
	}
	_ = s.Worktrees.Remove(ctx, s.Cfg.ProjectRoot, run.WorktreePath, run.WorktreeBranch)
}

// CleanupSessionWorktrees removes all worktrees associated with a parent session.
func (s *Service) CleanupSessionWorktrees(ctx context.Context, parentSessionID string) {
	if s.Worktrees == nil || s.Store == nil || parentSessionID == "" {
		return
	}
	runs, err := s.Store.ListRuns(ctx, parentSessionID)
	if err != nil {
		logging.L().Warn("list runs for worktree cleanup failed", zap.String("session_id", parentSessionID), zap.Error(err))
		return
	}
	seen := make(map[string]bool)
	for _, run := range runs {
		if run.WorktreePath == "" || seen[run.WorktreePath] {
			continue
		}
		seen[run.WorktreePath] = true
		if err := s.Worktrees.ValidatePath(run.WorktreePath); err != nil {
			logging.L().Warn("skip worktree remove: path outside base dir",
				zap.String("path", run.WorktreePath),
				zap.Error(err),
			)
			continue
		}
		_ = s.Worktrees.Remove(ctx, s.Cfg.ProjectRoot, run.WorktreePath, run.WorktreeBranch)
	}
}

// CleanupExpiredWorktrees removes worktree directories older than worktree_ttl.
func (s *Service) CleanupExpiredWorktrees(ctx context.Context) {
	if s.Worktrees == nil {
		return
	}
	entries, err := os.ReadDir(s.Worktrees.BaseDir)
	if err != nil {
		if !os.IsNotExist(err) {
			logging.L().Warn("read worktree base dir failed", zap.Error(err))
		}
		return
	}
	ttl := s.worktreeTTL()
	now := time.Now()
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(s.Worktrees.BaseDir, ent.Name())
		age := now.Sub(info.ModTime())
		if age < ttl {
			continue
		}
		branch := ""
		if name := ent.Name(); strings.HasPrefix(name, "wt-") {
			branch = "ds-code/agent-" + strings.TrimPrefix(name, "wt-")
		}
		_ = s.Worktrees.Remove(ctx, s.Cfg.ProjectRoot, path, branch)
		logging.L().Info("expired worktree removed", zap.String("path", path), zap.Duration("age", age))
	}
}
