package patch

import (
	"fmt"
	"strings"
)

const (
	beginMarker  = "*** Begin Patch"
	endMarker    = "*** End Patch"
	addMarker    = "*** Add File: "
	deleteMarker = "*** Delete File: "
	updateMarker = "*** Update File: "
	moveMarker   = "*** Move to: "
	eofMarker    = "*** End of File"
)

// FileChange is one file operation in a Codex-style patch.
type FileChange struct {
	Kind     ChangeKind
	Path     string
	MoveTo   string
	AddLines []string
	Chunks   []Chunk
}

// Chunk is a replace region in an update hunk.
type Chunk struct {
	Context string
	Old     []string
	New     []string
	EOF     bool
}

// Parse parses a Codex apply_patch document.
func Parse(text string) ([]FileChange, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "<<") {
		text = unwrapHeredoc(text)
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty patch")
	}
	start, end := 0, len(lines)-1
	if strings.Contains(lines[0], beginMarker) {
		start = 1
	}
	if end >= start && strings.Contains(lines[end], endMarker) {
		end--
	}
	if start > end {
		return nil, fmt.Errorf("patch missing body")
	}
	body := lines[start : end+1]
	var out []FileChange
	i := 0
	for i < len(body) {
		line := strings.TrimSpace(body[i])
		if line == "" {
			i++
			continue
		}
		switch {
		case strings.HasPrefix(line, addMarker):
			path := strings.TrimSpace(strings.TrimPrefix(line, addMarker))
			i++
			var content []string
			for i < len(body) {
				l := body[i]
				if isHunkHeader(strings.TrimSpace(l)) {
					break
				}
				if strings.HasPrefix(l, "+") {
					content = append(content, strings.TrimPrefix(l, "+"))
				}
				i++
			}
			out = append(out, FileChange{Kind: ChangeAdd, Path: path, AddLines: content})
		case strings.HasPrefix(line, deleteMarker):
			path := strings.TrimSpace(strings.TrimPrefix(line, deleteMarker))
			out = append(out, FileChange{Kind: ChangeDelete, Path: path})
			i++
		case strings.HasPrefix(line, updateMarker):
			path := strings.TrimSpace(strings.TrimPrefix(line, updateMarker))
			i++
			ch := FileChange{Kind: ChangeUpdate, Path: path}
			for i < len(body) {
				l := strings.TrimSpace(body[i])
				if isHunkHeader(l) && !strings.HasPrefix(l, "@@") && !strings.HasPrefix(l, moveMarker) {
					break
				}
				if strings.HasPrefix(l, moveMarker) {
					ch.MoveTo = strings.TrimSpace(strings.TrimPrefix(l, moveMarker))
					i++
					continue
				}
				if l == eofMarker {
					if len(ch.Chunks) > 0 {
						ch.Chunks[len(ch.Chunks)-1].EOF = true
					}
					i++
					continue
				}
				if strings.HasPrefix(l, "@@") {
					ctx := strings.TrimSpace(strings.TrimPrefix(l, "@@"))
					i++
					var chunk Chunk
					if ctx != "" {
						chunk.Context = ctx
					}
					for i < len(body) {
						raw := body[i]
						trim := strings.TrimSpace(raw)
						if isHunkHeader(trim) && !strings.HasPrefix(trim, "@@") {
							break
						}
						if trim == eofMarker {
							chunk.EOF = true
							i++
							break
						}
						if strings.HasPrefix(trim, "@@") {
							break
						}
						if len(raw) == 0 {
							chunk.Old = append(chunk.Old, "")
							i++
							continue
						}
						switch raw[0] {
						case '-':
							chunk.Old = append(chunk.Old, raw[1:])
						case '+':
							chunk.New = append(chunk.New, raw[1:])
						case ' ':
							line := raw[1:]
							chunk.Old = append(chunk.Old, line)
							chunk.New = append(chunk.New, line)
						default:
							return nil, fmt.Errorf("invalid change line: %q", raw)
						}
						i++
					}
					ch.Chunks = append(ch.Chunks, chunk)
					continue
				}
				i++
			}
			out = append(out, ch)
		default:
			return nil, fmt.Errorf("unexpected line: %q", line)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no file operations in patch")
	}
	return out, nil
}

func isHunkHeader(line string) bool {
	return strings.HasPrefix(line, "*** ")
}

func unwrapHeredoc(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "<<") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
	}
	s = strings.TrimSuffix(s, "EOF")
	return strings.TrimSpace(s)
}

// Paths returns all relative paths referenced by the patch.
func Paths(text string) ([]string, error) {
	changes, err := Parse(text)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var paths []string
	for _, c := range changes {
		if c.Path != "" && !seen[c.Path] {
			seen[c.Path] = true
			paths = append(paths, c.Path)
		}
		if c.MoveTo != "" && !seen[c.MoveTo] {
			seen[c.MoveTo] = true
			paths = append(paths, c.MoveTo)
		}
	}
	return paths, nil
}

// CountChangedLines estimates + and - lines for budget checks.
func CountChangedLines(text string) (int, error) {
	changes, err := Parse(text)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, c := range changes {
		n += len(c.AddLines)
		for _, ch := range c.Chunks {
			n += len(ch.Old) + len(ch.New)
		}
		if c.Kind == ChangeDelete {
			n++
		}
	}
	return n, nil
}
