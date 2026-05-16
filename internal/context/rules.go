package context

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadRules reads all .md files from .ds-code/rules/ under projectRoot.
func LoadRules(projectRoot string) (string, error) {
	dir := filepath.Join(projectRoot, ".ds-code", "rules")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("### ")
		b.WriteString(strings.TrimSuffix(name, ".md"))
		b.WriteString("\n")
		b.Write(data)
	}
	return strings.TrimSpace(b.String()), nil
}
