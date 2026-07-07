package checkpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

// Store persists checkpoints per session under the project checkpoint dir.
type Store struct {
	root string
}

// OpenAt creates a checkpoint store at an explicit root directory.
func OpenAt(rootDir string) (*Store, error) {
	if rootDir == "" {
		return nil, fmt.Errorf("checkpoint: empty root dir")
	}
	if err := os.MkdirAll(rootDir, 0o700); err != nil {
		return nil, fmt.Errorf("checkpoint: mkdir: %w", err)
	}
	return &Store{root: rootDir}, nil
}

// OpenStore creates the checkpoint store for a project.
func OpenStore(projectRoot string) (*Store, error) {
	if _, err := config.EnsureProjectDataDir(projectRoot); err != nil {
		return nil, err
	}
	dir := config.DefaultCheckpointDir(projectRoot)
	return OpenAt(dir)
}

func (s *Store) sessionDir(sessionID string) string {
	return filepath.Join(s.root, sessionID)
}

func (s *Store) recordPath(sessionID string, id int) string {
	return filepath.Join(s.sessionDir(sessionID), fmt.Sprintf("%04d.json", id))
}

// Create saves a new checkpoint and returns it with an assigned ID.
func (s *Store) Create(_ context.Context, sessionID, tool string, files []FileState, patch string) (Record, error) {
	if sessionID == "" {
		return Record{}, fmt.Errorf("checkpoint: session id required")
	}
	dir := s.sessionDir(sessionID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Record{}, err
	}
	id, err := s.nextID(dir)
	if err != nil {
		return Record{}, err
	}
	rec := Record{
		ID:        id,
		SessionID: sessionID,
		Tool:      tool,
		Files:     files,
		Patch:     patch,
		CreatedAt: time.Now().UTC(),
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return Record{}, err
	}
	path := s.recordPath(sessionID, id)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return Record{}, err
	}
	return rec, nil
}

func (s *Store) nextID(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	max := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSuffix(e.Name(), ".json"))
		if err == nil && n > max {
			max = n
		}
	}
	return max + 1, nil
}

// List returns checkpoint metadata for a session, oldest first.
func (s *Store) List(_ context.Context, sessionID string) ([]Meta, error) {
	dir := s.sessionDir(sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var metas []Meta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		rec, err := s.loadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		metas = append(metas, metaFromRecord(rec))
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].ID < metas[j].ID })
	return metas, nil
}

// Get loads a checkpoint by ID.
func (s *Store) Get(_ context.Context, sessionID string, id int) (Record, error) {
	path := s.recordPath(sessionID, id)
	return s.loadFile(path)
}

func (s *Store) loadFile(path string) (Record, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	var rec Record
	if err := json.Unmarshal(b, &rec); err != nil {
		return Record{}, err
	}
	return rec, nil
}

func metaFromRecord(rec Record) Meta {
	files := make([]string, 0, len(rec.Files))
	for _, f := range rec.Files {
		files = append(files, f.RelPath)
	}
	return Meta{
		ID:        rec.ID,
		Tool:      rec.Tool,
		Files:     files,
		CreatedAt: rec.CreatedAt,
	}
}
