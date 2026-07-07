package config

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
)

// SavePermissionMode writes permission.mode to user or project config atomically.
func SavePermissionMode(projectRoot string, isProject bool, mode permissionmode.Mode) error {
	if !mode.Configured() {
		return fmt.Errorf("config: invalid permission mode %q", mode)
	}
	var path string
	if isProject {
		path = ProjectConfigPath(projectRoot)
	} else {
		p, err := UserConfigPath()
		if err != nil {
			return err
		}
		path = p
	}
	return withConfigFileLock(path, func() error {
		doc, err := readYAMLDocument(path)
		if err != nil {
			return err
		}
		if err := setPermissionModeNode(doc, string(mode)); err != nil {
			return err
		}
		return writeYAMLDocumentAtomic(path, doc)
	})
}

func setPermissionModeNode(doc *yamlDocument, mode string) error {
	mapping := mappingNode(doc)
	if mapping == nil {
		return fmt.Errorf("config: invalid yaml document root")
	}
	_, permVal := findMappingPair(mapping, "permission")
	if permVal == nil {
		permKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "permission"}
		permVal = &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{}}
		mapping.Content = append(mapping.Content, permKey, permVal)
	}
	if permVal.Kind != yaml.MappingNode {
		return fmt.Errorf("config: permission must be a mapping")
	}
	_, modeVal := findMappingPair(permVal, "mode")
	if modeVal == nil {
		modeKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "mode"}
		modeVal = &yaml.Node{Kind: yaml.ScalarNode, Value: mode}
		permVal.Content = append(permVal.Content, modeKey, modeVal)
		return nil
	}
	modeVal.Value = mode
	return nil
}
