package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
)

const registryVersion = 1

// WindowLayout persists column widths and collapse state.
type WindowLayout struct {
	Width          int  `json:"width,omitempty"`
	Height         int  `json:"height,omitempty"`
	LeftWidth      int  `json:"leftWidth,omitempty"`
	RightWidth     int  `json:"rightWidth,omitempty"`
	LeftCollapsed  bool `json:"leftCollapsed,omitempty"`
	RightCollapsed bool `json:"rightCollapsed,omitempty"`
}

// Entry is one workspace in the registry.
type Entry struct {
	ID           string `json:"id"`
	Root         string `json:"root"`
	Name         string `json:"name"`
	AddedAt      int64  `json:"addedAt"`
	LastOpenedAt int64  `json:"lastOpenedAt"`
}

// RegistryFile is the on-disk workspaces.json schema.
type RegistryFile struct {
	V          int          `json:"v"`
	Active     string       `json:"active"`
	Workspaces []Entry      `json:"workspaces"`
	Window     WindowLayout `json:"window"`
}

// Registry loads and saves ~/.ds-code/desktop/workspaces.json.
type Registry struct {
	path string
	data RegistryFile
}

// DefaultRegistryPath returns ~/.ds-code/desktop/workspaces.json.
func DefaultRegistryPath() (string, error) {
	home, err := datadir.UserDataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "desktop", "workspaces.json"), nil
}

// LoadRegistry reads the registry from disk (empty if missing).
func LoadRegistry() (*Registry, error) {
	path, err := DefaultRegistryPath()
	if err != nil {
		return nil, err
	}
	return LoadRegistryAt(path)
}

// LoadRegistryAt reads the registry from an explicit path (for tests).
func LoadRegistryAt(path string) (*Registry, error) {
	r := &Registry{path: path}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			r.data = RegistryFile{V: registryVersion}
			return r, nil
		}
		return nil, fmt.Errorf("workspace registry: read: %w", err)
	}
	if err := json.Unmarshal(b, &r.data); err != nil {
		return nil, fmt.Errorf("workspace registry: parse: %w", err)
	}
	if r.data.V == 0 {
		r.data.V = registryVersion
	}
	return r, nil
}

// Data returns a copy of the current registry state.
func (r *Registry) Data() RegistryFile {
	return r.data
}

// Active returns the active workspace id.
func (r *Registry) Active() string {
	return r.data.Active
}

// SetActive updates the active workspace id.
func (r *Registry) SetActive(id string) {
	r.data.Active = id
}

// WindowLayout returns persisted window layout.
func (r *Registry) WindowLayout() WindowLayout {
	return r.data.Window
}

// SetWindowLayout updates persisted window layout.
func (r *Registry) SetWindowLayout(w WindowLayout) {
	r.data.Window = w
}

// FindByRoot returns an entry index for the absolute project root, or -1.
func (r *Registry) FindByRoot(absRoot string) int {
	for i, e := range r.data.Workspaces {
		if e.Root == absRoot {
			return i
		}
	}
	return -1
}

// Upsert adds or updates a workspace entry (dedupe by root).
func (r *Registry) Upsert(e Entry) {
	now := time.Now().Unix()
	if e.AddedAt == 0 {
		e.AddedAt = now
	}
	e.LastOpenedAt = now
	if i := r.FindByRoot(e.Root); i >= 0 {
		r.data.Workspaces[i].Name = e.Name
		r.data.Workspaces[i].LastOpenedAt = e.LastOpenedAt
		if e.ID != "" {
			r.data.Workspaces[i].ID = e.ID
		}
		return
	}
	r.data.Workspaces = append(r.data.Workspaces, e)
}

// Remove deletes a workspace entry by id (does not touch project data on disk).
func (r *Registry) Remove(id string) bool {
	for i, e := range r.data.Workspaces {
		if e.ID == id {
			r.data.Workspaces = append(r.data.Workspaces[:i], r.data.Workspaces[i+1:]...)
			if r.data.Active == id {
				r.data.Active = ""
				if len(r.data.Workspaces) > 0 {
					r.data.Active = r.data.Workspaces[0].ID
				}
			}
			return true
		}
	}
	return false
}

// Save atomically writes the registry (mode 0600).
func (r *Registry) Save() error {
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("workspace registry: mkdir: %w", err)
	}
	r.data.V = registryVersion
	b, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("workspace registry: write tmp: %w", err)
	}
	if err := os.Rename(tmp, r.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("workspace registry: rename: %w", err)
	}
	return nil
}
