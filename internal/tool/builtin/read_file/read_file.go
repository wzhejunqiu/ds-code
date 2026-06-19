package read_file

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/textfile"
	"go.uber.org/zap"
)

// ReadFileTool reads file contents with optional line range.
type ReadFileTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	Strict bool
}

func (t *ReadFileTool) Name() string { return tool.NameReadFile.String() }

func (t *ReadFileTool) IsReadOnly() bool        { return true }
func (t *ReadFileTool) IsConcurrencySafe() bool { return true }

func (t *ReadFileTool) Description() string {
	return fmt.Sprintf(DescReadFile, tool.NameShell.String())
}

func (t *ReadFileTool) Schema() map[string]any {
	maxLines := t.configuredMaxLines()
	return tool.ObjectSchema(map[string]any{
		"path":   map[string]any{"type": "string", "description": builtin.SchemaPathFileRelOrAbs},
		"offset": map[string]any{"type": "integer", "description": SchemaOffset},
		"limit":  map[string]any{"type": "integer", "description": fmt.Sprintf(SchemaLimitFmt, maxLines)},
	}, []string{"path"}, t.Strict)
}

func (t *ReadFileTool) configuredMaxLines() int {
	maxLines := t.Cfg.Tools.ReadFile.MaxLines
	if maxLines <= 0 {
		maxLines = 2000
	}
	return maxLines
}

func (t *ReadFileTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *ReadFileTool) WithPerm(perm *permission.Engine) tool.Tool {
	cp := *t
	cp.Perm = perm
	return &cp
}

func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var in struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if in.Path == "" {
		return "", fmt.Errorf("%s", builtin.ErrPathRequired)
	}
	abs, err := t.Perm.CheckReadablePath(in.Path)
	if err != nil {
		return "", err
	}

	maxLines := t.configuredMaxLines()
	maxBytes := t.Cfg.Tools.ReadFile.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 2 << 20
	}

	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if st.Size() > int64(maxBytes) {
		return "", fmt.Errorf(ErrFileTooLarge, st.Size(), maxBytes)
	}

	if !t.Perm.IsProjectDataPath(abs) && !textfile.IsTextFile(abs) {
		logging.L().Info("read_file skipped non-text file",
			zap.String("path", in.Path),
			zap.String("abs", abs),
		)
		return "", fmt.Errorf(ErrNotTextFile, in.Path)
	}

	start, end, rangeTruncated, err := resolveReadOffsetLimit(in.Offset, in.Limit, maxLines)
	if err != nil {
		return "", err
	}

	return formatReadFileOutput(abs, start, end, rangeTruncated, maxLines)
}

func resolveReadOffsetLimit(offset, limit, maxLines int) (readStart, readEnd int, truncated bool, err error) {
	if offset < 0 || limit < 0 {
		return 0, 0, false, fmt.Errorf("%s", builtin.ErrOffsetLimitNonNegative)
	}
	readStart = 1
	if offset > 0 {
		readStart = offset
	}
	readLimit := limit
	if readLimit <= 0 {
		readLimit = maxLines
	}
	if readLimit > maxLines {
		readLimit = maxLines
		truncated = limit > maxLines
	}
	readEnd = readStart + readLimit - 1
	return readStart, readEnd, truncated, nil
}

func formatReadFileOutput(abs string, start, end int, rangeTruncated bool, maxLines int) (string, error) {
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
		return fmt.Sprintf(ResultEmptyOffsetBeyond, start, lineNo), nil
	}
	if rangeTruncated {
		fmt.Fprintf(&out, MsgTruncatedMaxLines, maxLines)
	}
	if moreAfter > 0 {
		fmt.Fprintf(&out, MsgMoreLinesNotShown, moreAfter)
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

var _ tool.Tool = (*ReadFileTool)(nil)
