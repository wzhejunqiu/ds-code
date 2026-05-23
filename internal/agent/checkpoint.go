package agent

import (
	"context"
	"fmt"

	"github.com/hejunqiu/ds-code/internal/checkpoint"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"go.uber.org/zap"
)

func (r *Runner) recordCheckpoint(ctx context.Context, sessionID, toolName string, args map[string]any) error {
	if r.Checkpoints == nil || sessionID == "" {
		return nil
	}
	if !isCheckpointTool(toolName) {
		return nil
	}
	paths, err := checkpoint.PathsFromTool(toolName, r.Perm.Workspace, args)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return nil
	}
	files, err := checkpoint.CapturePaths(r.Perm.Workspace, r.Perm.ResolvePath, paths)
	if err != nil {
		return err
	}
	patch, _ := args["patch"].(string)
	rec, err := r.Checkpoints.Create(ctx, sessionID, toolName, files, patch)
	if err != nil {
		return err
	}
	var totalBytes int
	for _, f := range files {
		totalBytes += len(f.Content)
	}
	logging.L().Debug("checkpoint created",
		zap.String("session_id", sessionID),
		zap.String("tool", toolName),
		zap.Int("checkpoint_id", rec.ID),
		zap.Int("paths", len(files)),
		zap.Int("bytes", totalBytes),
	)
	return nil
}

func isCheckpointTool(name string) bool {
	switch name {
	case "apply_patch", "write_file":
		return true
	default:
		return false
	}
}

// RewindCheckpoint restores workspace state and appends a system event message.
func (r *Runner) RewindCheckpoint(ctx context.Context, sessionID string, id int) error {
	if r.Checkpoints == nil {
		return fmt.Errorf("checkpoint store not configured")
	}
	rec, err := r.Checkpoints.Get(ctx, sessionID, id)
	if err != nil {
		return err
	}
	if err := checkpoint.ApplyRewind(r.Perm.Workspace, rec); err != nil {
		return err
	}
	return r.Sessions.AppendMessage(ctx, session.Message{
		SessionID: sessionID,
		Role:      role.System,
		Content:   fmt.Sprintf("[ds-code] Rewound workspace to checkpoint #%d (%s)", id, rec.Tool),
	})
}
