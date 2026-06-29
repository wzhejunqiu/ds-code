package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
)

const (
	memoryMaxChars    = 7000  // ~2000 tokens at 3.5 chars/token (LoadAgentMemory injection cap)
	memorySlotMaxSize = 32768 // max bytes per slot file after append
)

var memorySlots = []string{"user.md", "feedback.md", "project.md", "reference.md"}

// AgentMemoryDir returns ~/.ds-code/agent-memory/{agentType}/.
func AgentMemoryDir(agentType string) (string, error) {
	root, err := datadir.UserDataHome()
	if err != nil {
		return "", err
	}
	agentType = sanitizeAgentType(agentType)
	return filepath.Join(root, "agent-memory", agentType), nil
}

func sanitizeAgentType(agentType string) string {
	agentType = strings.TrimSpace(agentType)
	if agentType == "" {
		return "general-purpose"
	}
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, agentType)
}

type memoryFile struct {
	name    string
	content string
	modTime time.Time
}

// LoadAgentMemory reads up to 3 most recently modified memory files, truncated to memoryMaxChars.
func LoadAgentMemory(agentType string) (string, error) {
	dir, err := AgentMemoryDir(agentType)
	if err != nil {
		return "", err
	}
	var files []memoryFile
	for _, slot := range memorySlots {
		path := filepath.Join(dir, slot)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		files = append(files, memoryFile{name: slot, content: string(data), modTime: info.ModTime()})
	}
	if len(files) == 0 {
		return "", nil
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})
	if len(files) > 3 {
		files = files[:3]
	}
	var b strings.Builder
	total := 0
	for _, f := range files {
		chunk := fmt.Sprintf("### %s\n%s\n\n", f.name, strings.TrimSpace(f.content))
		if total+len(chunk) > memoryMaxChars {
			remain := memoryMaxChars - total
			if remain > 0 {
				b.WriteString(chunk[:remain])
			}
			break
		}
		b.WriteString(chunk)
		total += len(chunk)
	}
	return strings.TrimSpace(b.String()), nil
}

// SaveAgentMemory writes content to a memory slot file.
func SaveAgentMemory(agentType, slot, content string) error {
	slot = strings.TrimSpace(slot)
	valid := false
	for _, s := range memorySlots {
		if slot == s || slot == strings.TrimSuffix(s, ".md") {
			if !strings.HasSuffix(slot, ".md") {
				slot = slot + ".md"
			}
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid memory slot %q (want user|feedback|project|reference)", slot)
	}
	dir, err := AgentMemoryDir(agentType)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, slot)
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("memory content is empty")
	}
	var body []byte
	if existing, err := os.ReadFile(path); err == nil {
		prev := strings.TrimSpace(string(existing))
		if prev != "" {
			body = []byte(prev + "\n" + content + "\n")
		} else {
			body = []byte(content + "\n")
		}
	} else if os.IsNotExist(err) {
		body = []byte(content + "\n")
	} else {
		return err
	}
	if len(body) > memorySlotMaxSize {
		body = body[len(body)-memorySlotMaxSize:]
	}
	return os.WriteFile(path, body, 0o600)
}

// FormatAgentMemory wraps loaded memory for system prompt injection.
func FormatAgentMemory(agentType string) string {
	body, err := LoadAgentMemory(agentType)
	if err != nil || body == "" {
		return ""
	}
	return fmt.Sprintf("<agent-memory agent_type=%q>\n%s\n</agent-memory>", sanitizeAgentType(agentType), body)
}
