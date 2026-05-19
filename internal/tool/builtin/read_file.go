package builtin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// ReadFileTool reads file contents with optional line range.
type ReadFileTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	Strict bool
}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Description() string {
	return "Read a file under the project workspace. Optional start/end (1-based inclusive line range)."
}

func (t *ReadFileTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"path":  map[string]any{"type": "string", "description": "Path to the file (relative to project root, or absolute if under project root)"},
		"start": map[string]any{"type": "integer", "description": "Start line (1-based, inclusive)"},
		"end":   map[string]any{"type": "integer", "description": "End line (1-based, inclusive)"},
	}, []string{"path"}, t.Strict)
}

func (t *ReadFileTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var in struct {
		Path  string `json:"path"`
		Start int    `json:"start"`
		End   int    `json:"end"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if in.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	abs, err := t.Perm.CheckReadablePath(in.Path)
	if err != nil {
		return "", err
	}

	maxLines := t.Cfg.Tools.ReadFile.MaxLines
	if maxLines <= 0 {
		maxLines = 500
	}
	maxBytes := t.Cfg.Tools.ReadFile.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 2 << 20
	}

	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if st.Size() > int64(maxBytes) {
		return "", fmt.Errorf("read_file: file size %d exceeds limit %d bytes", st.Size(), maxBytes)
	}

	start, end, rangeTruncated, err := resolveReadLineRange(in.Start, in.End, maxLines)
	if err != nil {
		return "", err
	}

	return formatReadFileOutput(abs, start, end, rangeTruncated)
}

func resolveReadLineRange(start, end, maxLines int) (readStart, readEnd int, truncated bool, err error) {
	if start < 0 || end < 0 {
		return 0, 0, false, fmt.Errorf("start and end must be non-negative")
	}
	switch {
	case start == 0 && end == 0:
		readStart = 1
		readEnd = readStart + maxLines - 1
	case start > 0 && end == 0:
		readStart = start
		readEnd = readStart + maxLines - 1
	case start == 0 && end > 0:
		readStart = 1
		readEnd = end
	default:
		readStart = start
		readEnd = end
	}
	if readEnd < readStart {
		return 0, 0, false, fmt.Errorf("end (%d) must be >= start (%d)", readEnd, readStart)
	}
	if readEnd-readStart+1 > maxLines {
		readEnd = readStart + maxLines - 1
		truncated = true
	}
	return readStart, readEnd, truncated, nil
}

func formatReadFileOutput(abs string, start, end int, rangeTruncated bool) (string, error) {
	f, err := os.Open(abs)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	var out strings.Builder
	lineNo := 0
	moreAfter := 0

	for sc.Scan() {
		lineNo++
		if lineNo < start {
			continue
		}
		if lineNo > end {
			moreAfter++
			for sc.Scan() {
				moreAfter++
			}
			break
		}
		fmt.Fprintf(&out, "%d|%s\n", lineNo, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	if lineNo < start {
		return fmt.Sprintf("(empty: start %d beyond file length %d)", start, lineNo), nil
	}
	if rangeTruncated {
		out.WriteString("\n... truncated to max_lines; adjust start/end to continue")
	}
	if moreAfter > 0 {
		fmt.Fprintf(&out, "\n... %d more lines not shown", moreAfter)
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

var _ tool.Tool = (*ReadFileTool)(nil)
