package workspace

import (
	"context"
	"fmt"

	desktopcp "github.com/wzhejunqiu/ds-code/desktop/checkpoint"
	desktopinspect "github.com/wzhejunqiu/ds-code/desktop/inspect"
)

// ListCheckpoints returns checkpoint metadata for a session.
func (m *Manager) ListCheckpoints(wsID, sessionID string) ([]desktopcp.Meta, error) {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return nil, err
	}
	if rt.runner == nil || rt.runner.Checkpoints == nil {
		return nil, nil
	}
	return desktopcp.List(context.Background(), rt.runner.Checkpoints, sessionID)
}

// PreviewCheckpointRewind returns diffs for rewinding to a checkpoint.
func (m *Manager) PreviewCheckpointRewind(wsID, sessionID string, id int) ([]desktopinspect.PatchFileDiff, error) {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return nil, err
	}
	if rt.runner == nil || rt.runner.Checkpoints == nil {
		return nil, fmt.Errorf("checkpoint store unavailable")
	}
	root, err := m.ProjectRoot(wsID)
	if err != nil {
		return nil, err
	}
	rec, err := desktopcp.Get(context.Background(), rt.runner.Checkpoints, sessionID, id)
	if err != nil {
		return nil, err
	}
	return desktopcp.PreviewRewind(root, rec)
}

// RewindCheckpoint restores workspace files and emits a system notice.
func (m *Manager) RewindCheckpoint(wsID, sessionID string, id int) error {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return err
	}
	if rt.runner == nil || rt.runner.Checkpoints == nil {
		return fmt.Errorf("checkpoint store unavailable")
	}
	ctx := context.Background()
	rec, err := desktopcp.Get(ctx, rt.runner.Checkpoints, sessionID, id)
	if err != nil {
		return err
	}
	if err := rt.runner.RewindCheckpoint(ctx, sessionID, id); err != nil {
		return err
	}
	m.emitSystemNotice(wsID, desktopcp.FormatRewindNotice(rec.ID, rec.Tool, rec.CreatedAt))
	return nil
}

// CheckpointNewerIDs returns ids of checkpoints newer than the target.
func (m *Manager) CheckpointNewerIDs(wsID, sessionID string, targetID int) ([]int, error) {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return nil, err
	}
	if rt.runner == nil || rt.runner.Checkpoints == nil {
		return nil, nil
	}
	return desktopcp.NewerIDs(context.Background(), rt.runner.Checkpoints, sessionID, targetID)
}
