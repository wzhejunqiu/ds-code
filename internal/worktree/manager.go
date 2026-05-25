// Package worktree manages git worktrees for agent isolation.
package worktree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
)

var slugRE = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// CreateOptions configures sparse checkout and large-dir symlinks after worktree add.
type CreateOptions struct {
	SparsePaths []string
	SymlinkDirs []string
}

// Manager creates and removes agent worktrees under a base directory.
type Manager struct {
	BaseDir string
}

// NewManager returns a worktree manager rooted at baseDir (typically project data dir).
func NewManager(baseDir string) *Manager {
	return &Manager{BaseDir: baseDir}
}

// Create adds a git worktree and branch for an agent run.
func (m *Manager) Create(ctx context.Context, repoRoot, slug string, opts CreateOptions) (path, branch string, err error) {
	if err := ValidateSlug(slug); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(m.BaseDir, 0o700); err != nil {
		return "", "", err
	}
	path = filepath.Join(m.BaseDir, "wt-"+slug)
	branch = "ds-code/agent-" + slug
	if err := runGit(ctx, repoRoot, "worktree", "add", "-b", branch, path); err != nil {
		return "", "", fmt.Errorf("worktree add: %w", err)
	}
	if err := applySparseCheckout(ctx, path, opts.SparsePaths); err != nil {
		logging.L().Warn("worktree sparse-checkout failed", zap.String("path", path), zap.Error(err))
	}
	if err := linkLargeDirs(repoRoot, path, opts.SymlinkDirs); err != nil {
		logging.L().Warn("worktree symlink dirs failed", zap.String("path", path), zap.Error(err))
	}
	return path, branch, nil
}

func applySparseCheckout(ctx context.Context, wtPath string, paths []string) error {
	if len(paths) == 0 {
		paths = []string{"/*"}
	}
	if err := runGit(ctx, wtPath, "sparse-checkout", "init", "--cone"); err != nil {
		return err
	}
	args := append([]string{"sparse-checkout", "set"}, paths...)
	return runGit(ctx, wtPath, args...)
}

func linkLargeDirs(repoRoot, wtPath string, dirs []string) error {
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" || strings.Contains(dir, "..") {
			continue
		}
		src := filepath.Join(repoRoot, dir)
		info, err := os.Stat(src)
		if err != nil || !info.IsDir() {
			continue
		}
		dst := filepath.Join(wtPath, dir)
		if _, err := os.Lstat(dst); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.Symlink(src, dst); err != nil {
			return fmt.Errorf("symlink %s: %w", dir, err)
		}
	}
	return nil
}

// Remove deletes a worktree and its branch.
func (m *Manager) Remove(ctx context.Context, repoRoot, path, branch string) error {
	if path != "" {
		_ = runGit(ctx, repoRoot, "worktree", "remove", "--force", path)
	}
	if branch != "" {
		_ = runGit(ctx, repoRoot, "branch", "-D", branch)
	}
	return nil
}

// ValidateSlug ensures slug is safe for paths and branch names.
func ValidateSlug(slug string) error {
	slug = strings.TrimSpace(slug)
	if slug == "" || len(slug) > 64 {
		return fmt.Errorf("invalid worktree slug")
	}
	if slugRE.ReplaceAllString(slug, "") != slug {
		return fmt.Errorf("invalid worktree slug characters")
	}
	if strings.Contains(slug, "..") {
		return fmt.Errorf("invalid worktree slug")
	}
	return nil
}

// ValidatePath ensures path is under the manager base directory.
func (m *Manager) ValidatePath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	base, err := filepath.Abs(m.BaseDir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(base, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("worktree path outside base dir")
	}
	return nil
}

func runGit(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
