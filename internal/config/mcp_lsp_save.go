package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// SaveMCPServers writes mcp.servers to user or project config atomically.
func SaveMCPServers(projectRoot string, isProject bool, servers []MCPServerConfig) error {
	return patchConfigFile(projectRoot, isProject, func(root map[string]any) {
		mcp, _ := root["mcp"].(map[string]any)
		if mcp == nil {
			mcp = map[string]any{}
			root["mcp"] = mcp
		}
		mcp["servers"] = serversToAny(servers)
	})
}

// SaveLSPConfig writes lsp settings to user or project config atomically.
func SaveLSPConfig(projectRoot string, isProject bool, cfg LSPConfig) error {
	return patchConfigFile(projectRoot, isProject, func(root map[string]any) {
		root["lsp"] = lspToAny(cfg)
	})
}

func patchConfigFile(projectRoot string, isProject bool, patch func(map[string]any)) error {
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
		root, err := docToMap(doc)
		if err != nil {
			return err
		}
		patch(root)
		return writeMapAtomic(path, root)
	})
}

func docToMap(doc *yamlDocument) (map[string]any, error) {
	var root map[string]any
	b, err := yaml.Marshal(&doc.root)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	if root == nil {
		root = map[string]any{}
	}
	return root, nil
}

func writeMapAtomic(path string, root map[string]any) error {
	b, err := yaml.Marshal(root)
	if err != nil {
		return fmt.Errorf("config: marshal %s: %w", path, err)
	}
	var doc yamlDocument
	if err := yaml.Unmarshal(b, &doc.root); err != nil {
		return fmt.Errorf("config: parse merged %s: %w", path, err)
	}
	return writeYAMLDocumentAtomic(path, &doc)
}

func serversToAny(servers []MCPServerConfig) []any {
	out := make([]any, 0, len(servers))
	for _, s := range servers {
		item := map[string]any{
			"name":    s.Name,
			"command": s.Command,
		}
		if len(s.Args) > 0 {
			item["args"] = s.Args
		}
		if len(s.Env) > 0 {
			item["env"] = s.Env
		}
		out = append(out, item)
	}
	return out
}

func lspToAny(cfg LSPConfig) map[string]any {
	out := map[string]any{
		"enabled": cfg.Enabled,
	}
	if len(cfg.WarmupOnStart) > 0 {
		out["warmup_on_start"] = cfg.WarmupOnStart
	}
	if cfg.DiagnosticsTimeout > 0 {
		out["diagnostics_timeout"] = cfg.DiagnosticsTimeout.String()
	}
	if len(cfg.Servers) > 0 {
		servers := map[string]any{}
		for id, s := range cfg.Servers {
			entry := map[string]any{}
			if s.Command != "" {
				entry["command"] = s.Command
			}
			if len(s.Args) > 0 {
				entry["args"] = s.Args
			}
			if len(s.Extensions) > 0 {
				entry["extensions"] = s.Extensions
			}
			if len(s.Env) > 0 {
				entry["env"] = s.Env
			}
			if s.Disabled {
				entry["disabled"] = true
			}
			servers[id] = entry
		}
		out["servers"] = servers
	}
	return out
}
