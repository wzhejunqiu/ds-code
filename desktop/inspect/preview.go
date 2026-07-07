package inspect

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/patch/apply"
)

// PatchFileDiff is one file's before/after preview for Monaco diff.
type PatchFileDiff struct {
	Path     string `json:"path"`
	Original string `json:"original"`
	Modified string `json:"modified"`
	Language string `json:"language"`
}

// PreviewPatch parses apply_patch text and returns in-memory diffs without writing.
func PreviewPatch(wsRoot, patchText string) ([]PatchFileDiff, error) {
	resolve := func(rel string) (string, error) {
		return apply.ResolveWorkspacePath(wsRoot, rel)
	}
	previews, err := apply.Preview(wsRoot, patchText, resolve)
	if err != nil {
		return nil, err
	}
	out := make([]PatchFileDiff, 0, len(previews))
	for _, p := range previews {
		path := p.Path
		if idx := strings.Index(path, " → "); idx >= 0 {
			path = path[:idx]
		}
		out = append(out, PatchFileDiff{
			Path:     p.Path,
			Original: p.Original,
			Modified: p.Modified,
			Language: LanguageForPath(path),
		})
	}
	return out, nil
}

// FilePreviewResult is a read-only file slice for Inspector.
type FilePreviewResult struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
}

// ReadFilePreview reads a file under workspace with optional line range.
func ReadFilePreview(wsRoot, path string, offset, limit int) (FilePreviewResult, error) {
	abs, err := apply.ResolveWorkspacePath(wsRoot, path)
	if err != nil {
		return FilePreviewResult{}, err
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return FilePreviewResult{}, err
	}
	content := string(b)
	if offset > 0 || limit > 0 {
		lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
		if content == "" {
			lines = nil
		}
		start := offset
		if start < 0 {
			start = 0
		}
		if start >= len(lines) {
			content = ""
		} else {
			end := len(lines)
			if limit > 0 && start+limit < end {
				end = start + limit
			}
			content = strings.Join(lines[start:end], "\n")
			if end < len(lines) || strings.HasSuffix(string(b), "\n") {
				content += "\n"
			}
		}
	}
	return FilePreviewResult{
		Path:     path,
		Content:  content,
		Language: LanguageForPath(path),
	}, nil
}

// LanguageForPath maps file extension to Monaco language id.
func LanguageForPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".json":
		return "json"
	case ".md", ".markdown":
		return "markdown"
	case ".yaml", ".yml":
		return "yaml"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".css":
		return "css"
	case ".html", ".htm":
		return "html"
	case ".sh", ".bash":
		return "shell"
	case ".sql":
		return "sql"
	case ".toml":
		return "toml"
	case ".xml":
		return "xml"
	default:
		return "plaintext"
	}
}
