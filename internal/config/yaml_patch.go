package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type yamlDocument struct {
	root yaml.Node
}

func readYAMLDocument(path string) (*yamlDocument, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			doc := &yamlDocument{}
			doc.root.Kind = yaml.DocumentNode
			doc.root.Content = []*yaml.Node{{
				Kind:    yaml.MappingNode,
				Content: []*yaml.Node{},
			}}
			return doc, nil
		}
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	var doc yamlDocument
	if err := yaml.Unmarshal(b, &doc.root); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	if doc.root.Kind == 0 {
		doc.root.Kind = yaml.DocumentNode
	}
	if doc.root.Kind == yaml.DocumentNode && len(doc.root.Content) == 0 {
		doc.root.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	}
	return &doc, nil
}

func writeYAMLDocumentAtomic(path string, doc *yamlDocument) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", dir, err)
	}
	out, err := yaml.Marshal(&doc.root)
	if err != nil {
		return fmt.Errorf("config: marshal %s: %w", path, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return fmt.Errorf("config: write %s: %w", tmp, err)
	}
	f, err := os.Open(tmp)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("config: fsync %s: %w", tmp, err)
	}
	_ = f.Close()
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("config: rename %s: %w", path, err)
	}
	return nil
}

func mappingNode(doc *yamlDocument) *yaml.Node {
	if doc.root.Kind != yaml.DocumentNode || len(doc.root.Content) == 0 {
		return nil
	}
	m := doc.root.Content[0]
	if m.Kind != yaml.MappingNode {
		return nil
	}
	return m
}

func findMappingPair(mapping *yaml.Node, key string) (keyNode, valueNode *yaml.Node) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		k := mapping.Content[i]
		if k.Value == key {
			return k, mapping.Content[i+1]
		}
	}
	return nil, nil
}

func appendWebAllowlistNode(doc *yamlDocument, host string) (changed bool, err error) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false, fmt.Errorf("config: empty web allowlist host")
	}
	mapping := mappingNode(doc)
	if mapping == nil {
		return false, fmt.Errorf("config: invalid yaml document root")
	}
	_, webVal := findMappingPair(mapping, "web")
	if webVal == nil {
		webKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "web"}
		webVal = &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{}}
		mapping.Content = append(mapping.Content, webKey, webVal)
	}
	if webVal.Kind != yaml.MappingNode {
		return false, fmt.Errorf("config: web must be a mapping")
	}
	_, allowlistVal := findMappingPair(webVal, "allowlist")
	if allowlistVal == nil {
		allowKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "allowlist"}
		allowlistVal = &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{}}
		webVal.Content = append(webVal.Content, allowKey, allowlistVal)
	}
	if allowlistVal.Kind != yaml.SequenceNode {
		return false, fmt.Errorf("config: web.allowlist must be a sequence")
	}
	for _, item := range allowlistVal.Content {
		if strings.EqualFold(strings.TrimSpace(item.Value), host) {
			return false, nil
		}
	}
	allowlistVal.Content = append(allowlistVal.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: host,
		Style: yaml.DoubleQuotedStyle,
	})
	return true, nil
}
