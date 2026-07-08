package main

import (
	"fmt"

	desktopbridge "github.com/wzhejunqiu/ds-code/desktop/bridge"
	desktopcheckpoint "github.com/wzhejunqiu/ds-code/desktop/checkpoint"
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
	PermissionMode        string `json:"permissionMode"`
	Model                 string `json:"model"`
	ReasoningEffort       string `json:"reasoningEffort"`
	RunMode               string `json:"runMode"`
	AssistantOutputFormat string `json:"assistantOutputFormat"`
	TracingEnabled        bool   `json:"tracingEnabled"`
	TracingExporter       string `json:"tracingExporter"`
	TracingOTLPEndpoint   string `json:"tracingOtlpEndpoint"`
	ProjectRoot           string `json:"projectRoot,omitempty"`
}

// SettingsPatch is a partial settings update from the desktop UI.
type SettingsPatch struct {
	PermissionMode      string `json:"permissionMode,omitempty"`
	Model               string `json:"model,omitempty"`
	ReasoningEffort     string `json:"reasoningEffort,omitempty"`
	TracingEnabled      *bool  `json:"tracingEnabled,omitempty"`
	TracingExporter     string `json:"tracingExporter,omitempty"`
	TracingOTLPEndpoint string `json:"tracingOtlpEndpoint,omitempty"`
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
		PermissionMode:        cfg.Permission.Mode.String(),
		Model:                 cfg.LLM.Model,
		ReasoningEffort:       cfg.LLM.ReasoningEffort,
		RunMode:               cfg.RunMode.String(),
		AssistantOutputFormat: cfg.Desktop.AssistantOutputFormat,
		TracingEnabled:        cfg.Tracing.Enabled,
		TracingExporter:       cfg.Tracing.Exporter,
		TracingOTLPEndpoint:   cfg.Tracing.OTLPEndpoint,
		ProjectRoot:           cfg.ProjectRoot,
	}, nil
}

// SaveSettingsPatch updates general settings for user or project scope.
func (s *DesktopService) SaveSettingsPatch(scope, wsID string, patch SettingsPatch) error {
	projectRoot, isProject, err := s.scopeProjectRoot(scope, wsID)
	if err != nil {
		return err
	}
	return config.SaveSettingsPatch(projectRoot, isProject, config.SettingsPatch{
		PermissionMode:      patch.PermissionMode,
		Model:               patch.Model,
		ReasoningEffort:     patch.ReasoningEffort,
		TracingEnabled:      patch.TracingEnabled,
		TracingExporter:     patch.TracingExporter,
		TracingOTLPEndpoint: patch.TracingOTLPEndpoint,
	})
}

// SaveConfigPatch updates permission mode for user or project scope.
func (s *DesktopService) SaveConfigPatch(scope, wsID, permissionMode string) error {
	return s.SaveSettingsPatch(scope, wsID, SettingsPatch{PermissionMode: permissionMode})
}

func (s *DesktopService) scopeProjectRoot(scope, wsID string) (projectRoot string, isProject bool, err error) {
	if scope == "project" {
		root, err := s.mgr.ProjectRoot(wsID)
		if err != nil {
			return "", false, err
		}
		return root, true, nil
	}
	return "", false, nil
}

// GetAssistantOutputFormat returns the session-level assistant output format.
func (s *DesktopService) GetAssistantOutputFormat(wsID, sessionID string) (string, error) {
	return s.mgr.GetAssistantOutputFormat(wsID, sessionID)
}

// SetAssistantOutputFormat sets the session format for subsequent assistant replies.
func (s *DesktopService) SetAssistantOutputFormat(wsID, sessionID, format string) error {
	return s.mgr.SetAssistantOutputFormat(wsID, sessionID, format)
}

// SaveDesktopAssistantOutputFormat saves the user-level default assistant output format.
func (s *DesktopService) SaveDesktopAssistantOutputFormat(format string) error {
	return config.SaveDesktopAssistantOutputFormat(format)
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

// ListCheckpoints returns checkpoint metadata for a session.
func (s *DesktopService) ListCheckpoints(wsID, sessionID string) ([]desktopcheckpoint.Meta, error) {
	return s.mgr.ListCheckpoints(wsID, sessionID)
}

// PreviewCheckpointRewind returns diffs for rewinding to a checkpoint.
func (s *DesktopService) PreviewCheckpointRewind(wsID, sessionID string, id int) ([]desktopinspect.PatchFileDiff, error) {
	return s.mgr.PreviewCheckpointRewind(wsID, sessionID, id)
}

// RewindCheckpoint restores workspace files to a checkpoint.
func (s *DesktopService) RewindCheckpoint(wsID, sessionID string, id int) error {
	return s.mgr.RewindCheckpoint(wsID, sessionID, id)
}

// CheckpointNewerIDs returns checkpoint ids newer than targetID.
func (s *DesktopService) CheckpointNewerIDs(wsID, sessionID string, targetID int) ([]int, error) {
	return s.mgr.CheckpointNewerIDs(wsID, sessionID, targetID)
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

// SessionUsage returns cumulative token usage and cost for a chat session.
func (s *DesktopService) SessionUsage(wsID, sessionID string) (desktopworkspace.SessionUsageView, error) {
	return s.mgr.SessionUsage(wsID, sessionID)
}

// ServiceStatus returns MCP/LSP status for a workspace.
func (s *DesktopService) ServiceStatus(wsID string) (desktopworkspace.ServiceStatusView, error) {
	return s.mgr.ServiceStatus(wsID)
}

// GetMCPLSPConfig loads MCP/LSP config for user or project scope.
func (s *DesktopService) GetMCPLSPConfig(scope, wsID string) (desktopworkspace.MCPLSPConfigView, error) {
	return s.mgr.GetMCPLSPConfig(scope, wsID)
}

// SaveMCPLSPConfig writes MCP/LSP config atomically (JSON body).
func (s *DesktopService) SaveMCPLSPConfig(scope, wsID, configJSON string) error {
	return s.mgr.SaveMCPLSPConfig(scope, wsID, configJSON)
}

// ReloadWorkspaceServices reconnects MCP/LSP for a workspace.
func (s *DesktopService) ReloadWorkspaceServices(wsID string) error {
	return s.mgr.ReloadWorkspaceServices(wsID)
}

// SearchChats finds sessions by title or message content.
func (s *DesktopService) SearchChats(wsID, query string) ([]desktopworkspace.ChatSummary, error) {
	return s.mgr.SearchChats(wsID, query)
}

// Close releases all workspace resources.
func (s *DesktopService) Close() {
	s.mgr.Close()
}
