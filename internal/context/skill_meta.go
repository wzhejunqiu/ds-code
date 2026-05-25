package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillMeta is parsed frontmatter from SKILL.md.
type SkillMeta struct {
	ContextMode string // e.g. "fork"
}

// LoadSkillWithMeta loads SKILL.md and parses optional YAML frontmatter between --- lines.
func LoadSkillWithMeta(projectRoot, skillName string) (SkillMeta, string, error) {
	text, err := LoadSkill(projectRoot, skillName)
	if err != nil {
		return SkillMeta{}, "", err
	}
	meta, body := parseSkillFrontmatter(text)
	return meta, body, nil
}

// SkillPath returns the resolved SKILL.md path if it exists.
func SkillPath(projectRoot, skillName string) (string, error) {
	skillName = strings.TrimSpace(skillName)
	candidates := []string{
		filepath.Join(projectRoot, ".ds-code", "skills", skillName, "SKILL.md"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".ds-code", "skills", skillName, "SKILL.md"))
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("skill %q not found", skillName)
}

func parseSkillFrontmatter(text string) (SkillMeta, string) {
	text = strings.TrimPrefix(text, "\ufeff")
	if !strings.HasPrefix(text, "---") {
		return SkillMeta{}, text
	}
	rest := text[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return SkillMeta{}, text
	}
	fm := rest[:end]
	body := strings.TrimPrefix(rest[end+4:], "\n")
	var meta SkillMeta
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "context":
			meta.ContextMode = v
		}
	}
	return meta, body
}
