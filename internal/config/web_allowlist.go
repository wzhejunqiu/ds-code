package config

// AppendWebAllowlist appends host to project .ds-code/config.yaml web.allowlist (deduped).
func AppendWebAllowlist(projectRoot, host string) error {
	path := ProjectConfigPath(projectRoot)
	return withConfigFileLock(path, func() error {
		doc, err := readYAMLDocument(path)
		if err != nil {
			return err
		}
		changed, err := appendWebAllowlistNode(doc, host)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		return writeYAMLDocumentAtomic(path, doc)
	})
}
