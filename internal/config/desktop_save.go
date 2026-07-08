package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// SaveDesktopAssistantOutputFormat writes desktop.assistant_output_format to user config.
func SaveDesktopAssistantOutputFormat(format string) error {
	if !contentformatValid(format) {
		return fmt.Errorf("config: invalid desktop.assistant_output_format %q", format)
	}
	path, err := UserConfigPath()
	if err != nil {
		return err
	}
	return withConfigFileLock(path, func() error {
		doc, err := readYAMLDocument(path)
		if err != nil {
			return err
		}
		if err := setDesktopAssistantOutputFormatNode(doc, format); err != nil {
			return err
		}
		return writeYAMLDocumentAtomic(path, doc)
	})
}

func setDesktopAssistantOutputFormatNode(doc *yamlDocument, format string) error {
	mapping := mappingNode(doc)
	if mapping == nil {
		return fmt.Errorf("config: invalid yaml document root")
	}
	_, desktopVal := findMappingPair(mapping, "desktop")
	if desktopVal == nil {
		desktopKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "desktop"}
		desktopVal = &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{}}
		mapping.Content = append(mapping.Content, desktopKey, desktopVal)
	}
	if desktopVal.Kind != yaml.MappingNode {
		return fmt.Errorf("config: desktop must be a mapping")
	}
	_, fmtVal := findMappingPair(desktopVal, "assistant_output_format")
	if fmtVal == nil {
		fmtKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "assistant_output_format"}
		fmtVal = &yaml.Node{Kind: yaml.ScalarNode, Value: format}
		desktopVal.Content = append(desktopVal.Content, fmtKey, fmtVal)
		return nil
	}
	fmtVal.Value = format
	return nil
}
