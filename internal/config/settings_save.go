package config

import (
	"fmt"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
)

// SaveLLMSettings writes llm.model and llm.reasoning_effort atomically.
func SaveLLMSettings(projectRoot string, isProject bool, model, reasoningEffort string) error {
	if model != "" && !slices.Contains(allowedModels, model) {
		return fmt.Errorf("config: llm.model must be one of %v, got %q", allowedModels, model)
	}
	if reasoningEffort != "" && !slices.Contains(allowedEffort, reasoningEffort) {
		return fmt.Errorf("config: llm.reasoning_effort must be high or max, got %q", reasoningEffort)
	}
	path, err := settingsConfigPath(projectRoot, isProject)
	if err != nil {
		return err
	}
	return withConfigFileLock(path, func() error {
		doc, err := readYAMLDocument(path)
		if err != nil {
			return err
		}
		if model != "" {
			if err := setNestedScalar(doc, []string{"llm", "model"}, model); err != nil {
				return err
			}
		}
		if reasoningEffort != "" {
			if err := setNestedScalar(doc, []string{"llm", "reasoning_effort"}, reasoningEffort); err != nil {
				return err
			}
		}
		return writeYAMLDocumentAtomic(path, doc)
	})
}

// SaveTracingSettings writes tracing.enabled, exporter, and otlp_endpoint atomically.
func SaveTracingSettings(projectRoot string, isProject bool, enabled bool, exporter, otlpEndpoint string) error {
	if exporter != "" && exporter != "log" && exporter != "otlp" {
		return fmt.Errorf("config: tracing.exporter must be empty, log, or otlp, got %q", exporter)
	}
	if exporter == "otlp" && otlpEndpoint == "" {
		return fmt.Errorf("config: tracing.otlp_endpoint is required when tracing.exporter is otlp")
	}
	path, err := settingsConfigPath(projectRoot, isProject)
	if err != nil {
		return err
	}
	return withConfigFileLock(path, func() error {
		doc, err := readYAMLDocument(path)
		if err != nil {
			return err
		}
		if err := setNestedScalar(doc, []string{"tracing", "enabled"}, boolString(enabled)); err != nil {
			return err
		}
		if err := setNestedScalar(doc, []string{"tracing", "exporter"}, exporter); err != nil {
			return err
		}
		if err := setNestedScalar(doc, []string{"tracing", "otlp_endpoint"}, otlpEndpoint); err != nil {
			return err
		}
		return writeYAMLDocumentAtomic(path, doc)
	})
}

// SaveSettingsPatch atomically updates general settings fields that are non-empty/non-nil.
func SaveSettingsPatch(projectRoot string, isProject bool, patch SettingsPatch) error {
	if patch.PermissionMode == "" && patch.Model == "" && patch.ReasoningEffort == "" &&
		patch.TracingEnabled == nil && patch.TracingExporter == "" && patch.TracingOTLPEndpoint == "" {
		return fmt.Errorf("config: empty settings patch")
	}
	if patch.PermissionMode != "" {
		mode, err := permissionmode.Parse(patch.PermissionMode)
		if err != nil {
			return err
		}
		if !mode.Configured() {
			return fmt.Errorf("config: invalid permission mode %q", patch.PermissionMode)
		}
	}
	if patch.Model != "" && !slices.Contains(allowedModels, patch.Model) {
		return fmt.Errorf("config: llm.model must be one of %v, got %q", allowedModels, patch.Model)
	}
	if patch.ReasoningEffort != "" && !slices.Contains(allowedEffort, patch.ReasoningEffort) {
		return fmt.Errorf("config: llm.reasoning_effort must be high or max, got %q", patch.ReasoningEffort)
	}
	if patch.TracingExporter != "" && patch.TracingExporter != "log" && patch.TracingExporter != "otlp" {
		return fmt.Errorf("config: tracing.exporter must be empty, log, or otlp, got %q", patch.TracingExporter)
	}
	if patch.TracingExporter == "otlp" && patch.TracingOTLPEndpoint == "" {
		return fmt.Errorf("config: tracing.otlp_endpoint is required when tracing.exporter is otlp")
	}

	path, err := settingsConfigPath(projectRoot, isProject)
	if err != nil {
		return err
	}
	return withConfigFileLock(path, func() error {
		doc, err := readYAMLDocument(path)
		if err != nil {
			return err
		}
		if patch.PermissionMode != "" {
			if err := setPermissionModeNode(doc, patch.PermissionMode); err != nil {
				return err
			}
		}
		if patch.Model != "" {
			if err := setNestedScalar(doc, []string{"llm", "model"}, patch.Model); err != nil {
				return err
			}
		}
		if patch.ReasoningEffort != "" {
			if err := setNestedScalar(doc, []string{"llm", "reasoning_effort"}, patch.ReasoningEffort); err != nil {
				return err
			}
		}
		if patch.TracingEnabled != nil {
			if err := setNestedScalar(doc, []string{"tracing", "enabled"}, boolString(*patch.TracingEnabled)); err != nil {
				return err
			}
		}
		if patch.TracingExporter != "" || patch.TracingEnabled != nil {
			if err := setNestedScalar(doc, []string{"tracing", "exporter"}, patch.TracingExporter); err != nil {
				return err
			}
		}
		if patch.TracingOTLPEndpoint != "" || patch.TracingExporter == "otlp" {
			if err := setNestedScalar(doc, []string{"tracing", "otlp_endpoint"}, patch.TracingOTLPEndpoint); err != nil {
				return err
			}
		}
		return writeYAMLDocumentAtomic(path, doc)
	})
}

// SettingsPatch is a partial general-settings update for desktop UI.
type SettingsPatch struct {
	PermissionMode      string
	Model               string
	ReasoningEffort     string
	TracingEnabled      *bool
	TracingExporter     string
	TracingOTLPEndpoint string
}

// SaveGeneralSettings writes permission mode and optional LLM/tracing fields.
func SaveGeneralSettings(projectRoot string, isProject bool, mode permissionmode.Mode, model, reasoningEffort string, tracingEnabled *bool, tracingExporter, tracingOTLPEndpoint string) error {
	patch := SettingsPatch{
		Model:               model,
		ReasoningEffort:     reasoningEffort,
		TracingEnabled:      tracingEnabled,
		TracingExporter:     tracingExporter,
		TracingOTLPEndpoint: tracingOTLPEndpoint,
	}
	if mode != "" {
		patch.PermissionMode = mode.String()
	}
	return SaveSettingsPatch(projectRoot, isProject, patch)
}

func settingsConfigPath(projectRoot string, isProject bool) (string, error) {
	if isProject {
		return ProjectConfigPath(projectRoot), nil
	}
	return UserConfigPath()
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func setNestedScalar(doc *yamlDocument, path []string, value string) error {
	if len(path) == 0 {
		return fmt.Errorf("config: empty yaml path")
	}
	mapping := mappingNode(doc)
	if mapping == nil {
		return fmt.Errorf("config: invalid yaml document root")
	}
	node := mapping
	for i := 0; i < len(path)-1; i++ {
		key := path[i]
		_, child := findMappingPair(node, key)
		if child == nil {
			childKey := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
			child = &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{}}
			node.Content = append(node.Content, childKey, child)
		}
		if child.Kind != yaml.MappingNode {
			return fmt.Errorf("config: %q must be a mapping", key)
		}
		node = child
	}
	lastKey := path[len(path)-1]
	_, val := findMappingPair(node, lastKey)
	if val == nil {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: lastKey}
		val = &yaml.Node{Kind: yaml.ScalarNode, Value: value}
		node.Content = append(node.Content, keyNode, val)
		return nil
	}
	val.Value = value
	return nil
}
