package main

import (
	"fmt"

	desktopbridge "github.com/wzhejunqiu/ds-code/desktop/bridge"
	desktopinspect "github.com/wzhejunqiu/ds-code/desktop/inspect"
	desktopsys "github.com/wzhejunqiu/ds-code/desktop/sys"
	desktopworkspace "github.com/wzhejunqiu/ds-code/desktop/workspace"
	"github.com/wzhejunqiu/ds-code/internal/config"
)

// DesktopService exposes Wails bindings for the desktop MVP.
type DesktopService struct {
	mgr *desktopworkspace.Manager
}

func newDesktopService(emit func(desktopbridge.AgentEventEnvelope)) (*DesktopService, error) {
	mgr, err := desktopworkspace.NewManager(emit)
	if err != nil {
		return nil, err
	}
	return &DesktopService{mgr: mgr}, nil
}

// ListWorkspaces returns all registered workspaces.
func (s *DesktopService) ListWorkspaces() []desktopworkspace.Summary {
	return s.mgr.List()
}

// AddWorkspace adds a project root as a workspace.
func (s *DesktopService) AddWorkspace(root string) (desktopworkspace.Summary, error) {
	return s.mgr.Add(root)
}

// RemoveWorkspace removes a workspace from the registry.
func (s *DesktopService) RemoveWorkspace(id string) error {
	return s.mgr.Remove(id)
}

// SwitchWorkspace sets the active workspace.
func (s *DesktopService) SwitchWorkspace(id string) error {
	return s.mgr.Switch(id)
}

// ActiveWorkspaceID returns the active workspace id.
func (s *DesktopService) ActiveWorkspaceID() string {
	return s.mgr.ActiveID()
}

// ListChats lists agent conversation windows for a workspace.
func (s *DesktopService) ListChats(wsID string) ([]desktopworkspace.ChatSummary, error) {
	return s.mgr.ListChats(wsID)
}

// CreateChat creates a new agent conversation in a workspace.
func (s *DesktopService) CreateChat(wsID string) (desktopworkspace.ChatSummary, error) {
	return s.mgr.CreateChat(wsID)
}

// ResumeChat loads session history.
func (s *DesktopService) ResumeChat(wsID, sessionID string) ([]desktopworkspace.ChatMessage, desktopworkspace.ChatSummary, error) {
	return s.mgr.ResumeChat(wsID, sessionID)
}

// RenameChat renames a session.
func (s *DesktopService) RenameChat(wsID, sessionID, title string) error {
	return s.mgr.RenameChat(wsID, sessionID, title)
}

// SendMessage starts an agent turn or handles a slash command.
func (s *DesktopService) SendMessage(wsID, sessionID, text string) (desktopworkspace.SlashResult, error) {
	return s.mgr.SendMessage(wsID, sessionID, text)
}

// ExecuteSlash runs a slash command without starting a turn.
func (s *DesktopService) ExecuteSlash(wsID, sessionID, line string) (desktopworkspace.SlashResult, error) {
	return s.mgr.ExecuteSlash(wsID, sessionID, line)
}

// SetRunMode switches agent/plan mode for a session.
func (s *DesktopService) SetRunMode(wsID, sessionID, mode string) error {
	return s.mgr.SetRunMode(wsID, sessionID, mode)
}

// SessionRunMode returns the run mode for a session.
func (s *DesktopService) SessionRunMode(wsID, sessionID string) (string, error) {
	return s.mgr.SessionRunMode(wsID, sessionID)
}

// CancelTurn cancels the in-flight turn for a workspace.
func (s *DesktopService) CancelTurn(wsID string) error {
	return s.mgr.CancelTurn(wsID)
}

// TurnStatus returns idle/running/waiting_permission for a workspace.
func (s *DesktopService) TurnStatus(wsID string) string {
	return s.mgr.TurnStatus(wsID)
}

// ResolvePermission completes an inline approval card.
func (s *DesktopService) ResolvePermission(wsID, requestID, choice string) error {
	reg, err := s.mgr.PermissionRegistry(wsID)
	if err != nil {
		return err
	}
	if !reg.ResolveChoice(requestID, choice) {
		return fmt.Errorf("unknown permission request: %s", requestID)
	}
	desktopsys.PermissionWaiting(false)
	return nil
}

// GetWindowLayout returns persisted column layout.
func (s *DesktopService) GetWindowLayout() desktopworkspace.WindowLayout {
	return s.mgr.Registry().WindowLayout()
}

// SaveWindowLayout persists column layout.
func (s *DesktopService) SaveWindowLayout(layout desktopworkspace.WindowLayout) error {
	s.mgr.Registry().SetWindowLayout(layout)
	return s.mgr.SaveRegistry()
}

// APIKeyStatus reports whether an API key is configured via environment.
func (s *DesktopService) APIKeyStatus() (bool, string) {
	_, err := config.LoadAPIKey()
	if err != nil {
		return false, err.Error()
	}
	return true, ""
}

// ConfigView is a read-only config snapshot for the settings UI.
type ConfigView struct {
	PermissionMode string `json:"permissionMode"`
	Model          string `json:"model"`
	RunMode        string `json:"runMode"`
	ProjectRoot    string `json:"projectRoot,omitempty"`
}

// GetConfig returns user-level or project-level config for a workspace.
// scope: "user" or "project" (requires wsID for project).
func (s *DesktopService) GetConfig(scope, wsID string) (ConfigView, error) {
	var startDir string
	if scope == "project" {
		root, err := s.mgr.ProjectRoot(wsID)
		if err != nil {
			return ConfigView{}, err
		}
		startDir = root
	}
	cfg, err := config.Load(nil, config.Options{
		StartDir:           startDir,
		RequireAPIKey:      false,
		SkipProjectDataDir: true,
	})
	if err != nil {
		return ConfigView{}, err
	}
	return ConfigView{
		PermissionMode: cfg.Permission.Mode.String(),
		Model:          cfg.LLM.Model,
		RunMode:        cfg.RunMode.String(),
		ProjectRoot:    cfg.ProjectRoot,
	}, nil
}

// SaveConfigPatch updates permission mode for user or project scope.
func (s *DesktopService) SaveConfigPatch(scope, wsID, permissionMode string) error {
	if permissionMode == "" {
		return fmt.Errorf("permissionMode required")
	}
	var startDir string
	if scope == "project" {
		root, err := s.mgr.ProjectRoot(wsID)
		if err != nil {
			return err
		}
		startDir = root
	}
	cfg, err := config.Load(nil, config.Options{
		StartDir:           startDir,
		RequireAPIKey:      false,
		SkipProjectDataDir: true,
	})
	if err != nil {
		return err
	}
	mode, err := parsePermissionMode(permissionMode)
	if err != nil {
		return err
	}
	var projectRoot string
	if scope == "project" {
		projectRoot = cfg.ProjectRoot
	}
	return savePermissionMode(projectRoot, scope == "project", mode)
}

// PickFolder opens a native folder picker (macOS); implemented in pick_darwin.go.
func (s *DesktopService) PickFolder() (string, error) {
	return pickFolderNative()
}

// OpenProject adds a workspace (compat with M0 CLI arg).
func (s *DesktopService) OpenProject(root string) error {
	_, err := s.mgr.Add(root)
	return err
}

// ProjectRoot returns the active workspace project root.
func (s *DesktopService) ProjectRoot() string {
	id := s.mgr.ActiveID()
	if id == "" {
		return ""
	}
	root, err := s.mgr.ProjectRoot(id)
	if err != nil {
		return ""
	}
	return root
}

// PreviewPatch returns Monaco-ready diffs for apply_patch text.
func (s *DesktopService) PreviewPatch(wsID, patchText string) ([]desktopinspect.PatchFileDiff, error) {
	root, err := s.mgr.ProjectRoot(wsID)
	if err != nil {
		return nil, err
	}
	return desktopinspect.PreviewPatch(root, patchText)
}

// ReadFilePreview returns read-only file content for Inspector.
func (s *DesktopService) ReadFilePreview(wsID, path string, offset, limit int) (desktopinspect.FilePreviewResult, error) {
	root, err := s.mgr.ProjectRoot(wsID)
	if err != nil {
		return desktopinspect.FilePreviewResult{}, err
	}
	return desktopinspect.ReadFilePreview(root, path, offset, limit)
}

// CheckDependencies reports git/node/gopls availability.
func (s *DesktopService) CheckDependencies() []desktopsys.DepStatus {
	return desktopsys.CheckDependencies()
}

// Close releases all workspace resources.
func (s *DesktopService) Close() {
	s.mgr.Close()
}
