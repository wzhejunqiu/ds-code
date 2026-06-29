package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wzhejunqiu/ds-code/internal/version"
	"gopkg.in/yaml.v3"
)

func rejectPermissionAuto(cmd *cobra.Command, projectRoot string, cfg *Config) error {
	if err := rejectAutoFromConfigYAML(projectRoot); err != nil {
		return err
	}
	return rejectAutoWithoutCLI(cmd, cfg)
}

func rejectAutoFromConfigYAML(projectRoot string) error {
	mode, found, err := readUserYAMLPermissionMode()
	if err != nil {
		return err
	}
	if found && mode == "auto" {
		return fmt.Errorf("config: user config.yaml cannot set permission.mode to auto; use --dangerously-auto or --permission-mode auto")
	}
	mode, found, err = readProjectYAMLPermissionMode(projectRoot)
	if err != nil {
		return err
	}
	if found && mode == "auto" {
		return fmt.Errorf("config: project %s/config.yaml cannot set permission.mode to auto; use --dangerously-auto or --permission-mode auto", version.UserDataDirName)
	}
	return nil
}

func rejectAutoWithoutCLI(cmd *cobra.Command, cfg *Config) error {
	if cfg.Permission.Mode == "auto" && !cliAllowsAuto(cmd) {
		return fmt.Errorf("permission.mode auto requires --dangerously-auto or --permission-mode auto")
	}
	return nil
}

func cliAllowsAuto(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	fs := cmd.PersistentFlags()
	if f := fs.Lookup("permission-mode"); f != nil && f.Changed && f.Value.String() == "auto" {
		return true
	}
	if f := fs.Lookup("dangerously-auto"); f != nil && f.Changed {
		v, err := fs.GetBool("dangerously-auto")
		return err == nil && v
	}
	return false
}

func readUserYAMLPermissionMode() (mode string, found bool, err error) {
	path, err := UserConfigPath()
	if err != nil {
		return "", false, err
	}
	return readYAMLPermissionMode(path)
}

func readProjectYAMLPermissionMode(projectRoot string) (mode string, found bool, err error) {
	return readYAMLPermissionMode(ProjectConfigPath(projectRoot))
}

func readYAMLPermissionMode(path string) (mode string, found bool, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return "", false, fmt.Errorf("config: read %s: %w", path, err)
	}
	perm, ok := doc["permission"].(map[string]any)
	if !ok {
		return "", false, nil
	}
	m, ok := perm["mode"].(string)
	if !ok || strings.TrimSpace(m) == "" {
		return "", false, nil
	}
	return strings.TrimSpace(m), true, nil
}
