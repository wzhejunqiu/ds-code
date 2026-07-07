package checkpoint

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/wzhejunqiu/ds-code/desktop/inspect"
	icp "github.com/wzhejunqiu/ds-code/internal/checkpoint"
	"github.com/wzhejunqiu/ds-code/internal/patch/apply"
)

// Meta is a checkpoint list row for the UI.
type Meta struct {
	ID        int      `json:"id"`
	Tool      string   `json:"tool"`
	Files     []string `json:"files"`
	CreatedAt int64    `json:"createdAt"`
}

// List returns checkpoint metadata for a session, oldest first.
func List(ctx context.Context, store *icp.Store, sessionID string) ([]Meta, error) {
	if store == nil {
		return nil, nil
	}
	metas, err := store.List(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	out := make([]Meta, 0, len(metas))
	for _, m := range metas {
		out = append(out, Meta{
			ID:        m.ID,
			Tool:      m.Tool,
			Files:     append([]string(nil), m.Files...),
			CreatedAt: m.CreatedAt.UnixMilli(),
		})
	}
	return out, nil
}

// PreviewRewind compares current workspace files with the checkpoint state.
// Original is current disk content; Modified is the state after rewind.
func PreviewRewind(wsRoot string, rec icp.Record) ([]inspect.PatchFileDiff, error) {
	out := make([]inspect.PatchFileDiff, 0, len(rec.Files))
	for _, f := range rec.Files {
		original, err := readCurrentContent(wsRoot, f.RelPath)
		if err != nil {
			return nil, err
		}
		var modified string
		if f.Existed {
			modified = string(f.Content)
		}
		if original == modified {
			continue
		}
		out = append(out, inspect.PatchFileDiff{
			Path:     f.RelPath,
			Original: original,
			Modified: modified,
			Language: inspect.LanguageForPath(f.RelPath),
		})
	}
	return out, nil
}

func readCurrentContent(wsRoot, relPath string) (string, error) {
	abs, err := apply.ResolveWorkspacePath(wsRoot, relPath)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

// Get loads a checkpoint record by id.
func Get(ctx context.Context, store *icp.Store, sessionID string, id int) (icp.Record, error) {
	return store.Get(ctx, sessionID, id)
}

// NewerIDs returns checkpoint ids greater than targetID for the same session.
func NewerIDs(ctx context.Context, store *icp.Store, sessionID string, targetID int) ([]int, error) {
	metas, err := store.List(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	var ids []int
	for _, m := range metas {
		if m.ID > targetID {
			ids = append(ids, m.ID)
		}
	}
	return ids, nil
}

// FormatRewindNotice builds a user-visible system notice after rewind.
func FormatRewindNotice(id int, tool string, at time.Time) string {
	return fmt.Sprintf("Rewound workspace to checkpoint #%d (%s, %s)", id, tool, at.Format("15:04"))
}
