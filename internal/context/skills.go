package context

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadSkill reads SKILL.md for the named skill from project or user dirs.
func LoadSkill(projectRoot, skillName string) (string, error) {
	skillName = strings.TrimSpace(skillName)
	if skillName == "" {
		return "", fmt.Errorf("skill name is required")
	}
	if strings.Contains(skillName, "..") || strings.ContainsAny(skillName, `/\`) {
		return "", fmt.Errorf("invalid skill name %q", skillName)
	}

	candidates := []string{
		filepath.Join(projectRoot, ".ds-code", "skills", skillName, "SKILL.md"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".ds-code", "skills", skillName, "SKILL.md"))
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("skill %q not found in .ds-code/skills/ or ~/.ds-code/skills/", skillName)
}

// ListSkillNames returns available skill directory names.
func ListSkillNames(projectRoot string) ([]string, error) {
	seen := make(map[string]struct{})
	var names []string
	add := func(root string) {
		dir := filepath.Join(root, "skills")
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skillPath := filepath.Join(dir, e.Name(), "SKILL.md")
			if _, err := os.Stat(skillPath); err != nil {
				continue
			}
			if _, ok := seen[e.Name()]; ok {
				continue
			}
			seen[e.Name()] = struct{}{}
			names = append(names, e.Name())
		}
	}
	add(filepath.Join(projectRoot, ".ds-code"))
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".ds-code"))
	}
	sort.Strings(names)
	return names, nil
}
