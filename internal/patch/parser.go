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
	Added   int // lines starting with '+'
	Removed int // lines starting with '-'
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
	seenPaths := make(map[string]struct{})
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
			if err := registerPath(seenPaths, path); err != nil {
				return nil, err
			}
			i++
			var content []string
			for i < len(body) {
				l := body[i]
				trim := strings.TrimSpace(l)
				if isHunkHeader(trim) {
					break
				}
				if trim == "" {
					i++
					continue
				}
				if strings.HasPrefix(l, "+") {
					content = append(content, strings.TrimPrefix(l, "+"))
				} else {
					return nil, fmt.Errorf("add file %s: line must start with '+': %q", path, l)
				}
				i++
			}
			out = append(out, FileChange{Kind: ChangeAdd, Path: path, AddLines: content})
		case strings.HasPrefix(line, deleteMarker):
			path := strings.TrimSpace(strings.TrimPrefix(line, deleteMarker))
			if err := registerPath(seenPaths, path); err != nil {
				return nil, err
			}
			out = append(out, FileChange{Kind: ChangeDelete, Path: path})
			i++
		case strings.HasPrefix(line, updateMarker):
			path := strings.TrimSpace(strings.TrimPrefix(line, updateMarker))
			if err := registerPath(seenPaths, path); err != nil {
				return nil, err
			}
			i++
			ch := FileChange{Kind: ChangeUpdate, Path: path}
			for i < len(body) {
				l := strings.TrimSpace(body[i])
				if isHunkHeader(l) && !strings.HasPrefix(l, "@@") && !strings.HasPrefix(l, moveMarker) {
					break
				}
				if strings.HasPrefix(l, moveMarker) {
					moveTo := strings.TrimSpace(strings.TrimPrefix(l, moveMarker))
					if err := registerPath(seenPaths, moveTo); err != nil {
						return nil, err
					}
					ch.MoveTo = moveTo
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
							chunk.Removed++
							i++
							continue
						}
						switch raw[0] {
						case '-':
							chunk.Old = append(chunk.Old, raw[1:])
							chunk.Removed++
						case '+':
							chunk.New = append(chunk.New, raw[1:])
							chunk.Added++
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
				if l != "" {
					return nil, fmt.Errorf("update %s: unexpected line: %q", path, body[i])
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
	if !strings.HasPrefix(s, "<<") {
		return s
	}
	idx := strings.Index(s, "\n")
	if idx < 0 {
		return strings.TrimSpace(strings.TrimPrefix(s, "<<"))
	}
	delimLine := strings.TrimSpace(s[:idx])
	delim := strings.TrimPrefix(delimLine, "<<")
	delim = strings.Trim(delim, "'\"")
	body := s[idx+1:]
	if delim != "" {
		body = strings.TrimSuffix(body, delim)
	}
	return strings.TrimSpace(body)
}

func registerPath(seen map[string]struct{}, path string) error {
	if path == "" {
		return fmt.Errorf("empty path in patch")
	}
	if _, ok := seen[path]; ok {
		return fmt.Errorf("duplicate path in patch: %s", path)
	}
	seen[path] = struct{}{}
	return nil
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
			n += ch.Added + ch.Removed
		}
		if c.Kind == ChangeDelete {
			n++
		}
	}
	return n, nil
}
