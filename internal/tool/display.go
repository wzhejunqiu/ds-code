package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/patch"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
)

var (
	grepFoundFilesRe       = regexp.MustCompile(`(?m)^Found (\d+) files$`)
	grepFoundOccurrencesRe = regexp.MustCompile(`(?m)^Found (\d+) occurrences across \d+ files$`)
)

const titleArgsMax = 80

// DisplayContext supplies MCP metadata for TUI tool display.
type DisplayContext struct {
	MCPLookup func(name string) (server string, ok bool)
}

// FromRegistry builds a DisplayContext from a tool registry.
func FromRegistry(reg *Registry) DisplayContext {
	if reg == nil {
		return DisplayContext{}
	}
	return DisplayContext{MCPLookup: reg.MCPServerForTool}
}

// IsMCPTool reports whether name is a registered MCP bare tool.
func (d DisplayContext) IsMCPTool(name string) bool {
	if d.MCPLookup == nil {
		return false
	}
	_, ok := d.MCPLookup(name)
	return ok
}

// MCPServerForTool returns the MCP server for a registered bare tool name.
func (d DisplayContext) MCPServerForTool(name string) (string, bool) {
	if d.MCPLookup == nil {
		return "", false
	}
	return d.MCPLookup(name)
}

// ApplyPatchFileDisplay is one per-file apply_patch TUI row.
type ApplyPatchFileDisplay struct {
	Filename string
	Added    int
	Removed  int
}

// ToolDisplayRow is one TUI tool block (args + optional command field).
type ToolDisplayRow struct {
	Args    string
	Command string
}

// ApplyPatchStarts returns per-file rows for apply_patch tool start, or nil for other tools.
func ApplyPatchStarts(name string, rawArgs []byte, workspace string) []ToolDisplayRow {
	if name != "apply_patch" {
		return nil
	}
	args := ArgsMap(rawArgs)
	p, _ := args["patch"].(string)
	if p == "" {
		return nil
	}
	displays := ApplyPatchFileDisplays(p, workspace)
	if len(displays) == 0 {
		return nil
	}
	rows := make([]ToolDisplayRow, len(displays))
	for i, d := range displays {
		rows[i] = ToolDisplayRow{Args: d.Filename, Command: EncodeApplyPatchStats(d.Added, d.Removed)}
	}
	return rows
}

// ToolEndRows returns per-file rows to finish for apply_patch, or nil for other tools.
func ToolEndRows(name string, rawArgs []byte, workspace string) []ToolDisplayRow {
	return ApplyPatchStarts(name, rawArgs, workspace)
}

// DisplaySummary formats tool arguments for the TUI (args line and optional command).
func DisplaySummary(name string, rawArgs []byte, workspace string, disp DisplayContext) (argsLine, command string) {
	args := ArgsMap(rawArgs)
	switch name {
	case NameShell.String():
		return ShellSummary(rawArgs)
	case "read_file":
		if p, _ := args["filepath"].(string); p != "" {
			return FormatReadFileDisplay(p, 0, 0), ""
		}
	case "write_file":
		if p, _ := args["path"].(string); p != "" {
			return FormatWriteFileDisplay(p), ""
		}
	case "grep":
		pat, _ := args["pattern"].(string)
		rel, _ := args["path"].(string)
		return FormatGrepDisplay(pat, rel, workspace), ""
	case "glob":
		pat, _ := args["pattern"].(string)
		rel, _ := args["path"].(string)
		return FormatGlobDisplay(pat, rel, workspace), ""
	case "list_dir":
		rel, _ := args["path"].(string)
		return FormatListDirDisplay(rel, workspace), ""
	case "apply_patch":
		if p, _ := args["patch"].(string); p != "" {
			displays := ApplyPatchFileDisplays(p, workspace)
			if len(displays) > 0 {
				d := displays[0]
				return d.Filename, EncodeApplyPatchStats(d.Added, d.Removed)
			}
		}
	case "agent", "task":
		if d, _ := args["description"].(string); d != "" {
			return FormatAgentDisplay(d), ""
		}
		if p, _ := args["prompt"].(string); p != "" {
			return FormatAgentDisplay(p), ""
		}
	case "web_fetch":
		if u, _ := args["url"].(string); u != "" {
			return FormatWebFetchDisplay(u), ""
		}
	default:
		if server, ok := disp.MCPServerForTool(name); ok {
			return FormatMCPCallDisplay(server, name, rawArgs), ""
		}
		if isLegacyMCPToolName(name) {
			return FormatMCPCallDisplayFromLegacy(name, rawArgs), ""
		}
	}
	if len(rawArgs) > 0 {
		return formatArgsJSON(rawArgs), ""
	}
	return "", ""
}

// ApplyPatchFileDisplays returns one display row per file in a patch.
func ApplyPatchFileDisplays(patchText, workspace string) []ApplyPatchFileDisplay {
	stats, err := patch.FileLineStats(patchText, nil)
	if err != nil || len(stats) == 0 {
		return nil
	}
	out := make([]ApplyPatchFileDisplay, 0, len(stats))
	for _, st := range stats {
		out = append(out, ApplyPatchFileDisplay{
			Filename: patch.DisplayBasename(st.Path),
			Added:    st.Added,
			Removed:  st.Removed,
		})
	}
	return out
}

// EncodeApplyPatchStats encodes +N/-M for the command field (0 = omit side).
func EncodeApplyPatchStats(added, removed int) string {
	if added <= 0 && removed <= 0 {
		return ""
	}
	a, r := "-", "-"
	if added > 0 {
		a = strconv.Itoa(added)
	}
	if removed > 0 {
		r = strconv.Itoa(removed)
	}
	return a + "|" + r
}

// DecodeApplyPatchStats decodes the command field from EncodeApplyPatchStats.
func DecodeApplyPatchStats(command string) (added, removed int, ok bool) {
	if command == "" {
		return 0, 0, false
	}
	parts := strings.SplitN(command, "|", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	if parts[0] != "-" {
		added, _ = strconv.Atoi(parts[0])
	}
	if parts[1] != "-" {
		removed, _ = strconv.Atoi(parts[1])
	}
	return added, removed, true
}

// displayPathLabel maps tool path args to a short UI label (never ".").
func displayPathLabel(relPath, workspace string) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" || relPath == "." || relPath == "/" {
		if workspace != "" {
			base := filepath.Base(workspace)
			if base != "" && base != "." {
				return base
			}
		}
		return "project"
	}
	base := filepath.Base(relPath)
	if base == "" || base == "." {
		return "project"
	}
	return base
}

// FormatGrepDisplay formats grep tool title.
func FormatGrepDisplay(pattern, relPath, workspace string) string {
	pattern = truncateOneLine(pattern, titleArgsMax)
	return "Grepped " + pattern + " in " + displayPathLabel(relPath, workspace)
}

// FormatGlobDisplay formats glob tool title.
func FormatGlobDisplay(pattern, relPath, workspace string) string {
	pattern = truncateOneLine(pattern, titleArgsMax)
	return "Searched files " + pattern + " in " + displayPathLabel(relPath, workspace)
}

// FormatListDirDisplay formats list_dir tool title.
func FormatListDirDisplay(relPath, workspace string) string {
	return "List " + displayPathLabel(relPath, workspace)
}

// FormatWriteFileDisplay formats write_file tool title.
func FormatWriteFileDisplay(path string) string {
	return "Write " + filepath.Base(path)
}

// FormatAgentDisplay formats the agent tool title.
func FormatAgentDisplay(descOrPrompt string) string {
	return "Agent: " + truncateOneLine(descOrPrompt, 120)
}

// FormatTaskDisplay is an alias for legacy task tool display.
func FormatTaskDisplay(descOrPrompt string) string {
	return FormatAgentDisplay(descOrPrompt)
}

// FormatWebFetchDisplay formats web_fetch tool title.
func FormatWebFetchDisplay(url string) string {
	return "Fetch " + truncateOneLine(url, 120)
}

func isLegacyMCPToolName(name string) bool {
	return strings.HasPrefix(name, "mcp__")
}

// FormatMCPCallDisplay formats MCP server, tool name, and compact arguments for the TUI.
func FormatMCPCallDisplay(server, toolName string, rawArgs []byte) string {
	title := FormatMCPBareDisplay(server, toolName)
	return appendMCPArgsTitle(title, rawArgs)
}

// FormatMCPCallDisplayFromLegacy formats legacy mcp__ tool names with JSON args.
func FormatMCPCallDisplayFromLegacy(toolName string, rawArgs []byte) string {
	title := FormatMCPDisplay(toolName)
	return appendMCPArgsTitle(title, rawArgs)
}

func appendMCPArgsTitle(title string, rawArgs []byte) string {
	if len(rawArgs) == 0 || bytes.Equal(bytes.TrimSpace(rawArgs), []byte("null")) {
		return title
	}
	compact := formatArgsJSON(rawArgs)
	if compact == "" || compact == "{}" {
		return title
	}
	return title + " " + compact
}

// FormatMCPBareDisplay formats a registered MCP bare tool for the TUI.
func FormatMCPBareDisplay(server, toolName string) string {
	if server != "" && toolName != "" {
		return "MCP " + server + " · " + toolName
	}
	return "MCP " + truncateOneLine(toolName, 80)
}

// FormatMCPDisplay formats a legacy mcp__ tool name for the TUI.
func FormatMCPDisplay(toolName string) string {
	server, toolPart, ok := parseMCPToolName(toolName)
	if ok && server != "" && toolPart != "" {
		return "MCP " + server + " · " + toolPart
	}
	raw := strings.TrimPrefix(toolName, "mcp__")
	if raw == "" {
		raw = toolName
	}
	return "MCP " + truncateOneLine(raw, 80)
}

func parseMCPToolName(name string) (server, tool string, ok bool) {
	if !strings.HasPrefix(name, "mcp__") {
		return "", "", false
	}
	rest := strings.TrimPrefix(name, "mcp__")
	parts := strings.SplitN(rest, "__", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ReadFileLineRange parses "N|..." lines from read_file tool output.
func ReadFileLineRange(result string) (start, end int, ok bool) {
	for _, line := range strings.Split(result, "\n") {
		i := strings.IndexByte(line, '|')
		if i <= 0 {
			continue
		}
		n, err := strconv.Atoi(line[:i])
		if err != nil || n <= 0 {
			continue
		}
		if !ok {
			start, end = n, n
			ok = true
			continue
		}
		if n < start {
			start = n
		}
		if n > end {
			end = n
		}
	}
	return start, end, ok
}

// FormatReadFileDisplay formats a human-readable read_file label for the TUI.
func FormatReadFileDisplay(path string, start, end int) string {
	line := "Read " + filepath.Base(path)
	if start > 0 && end >= start {
		line += fmt.Sprintf(" L%d-%d", start, end)
	}
	return line
}

// AppendReadFileLineRange updates a read_file display line with the actual line range.
func AppendReadFileLineRange(argsLine string, start, end int) string {
	path := readFilePathFromDisplay(argsLine)
	if path == "" {
		return argsLine
	}
	return FormatReadFileDisplay(path, start, end)
}

func readFilePathFromDisplay(line string) string {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "Read ") {
		rest := strings.TrimPrefix(line, "Read ")
		if i := strings.Index(rest, " L"); i >= 0 {
			rest = rest[:i]
		}
		return strings.TrimSpace(rest)
	}
	if strings.HasPrefix(line, "path=") {
		return strings.TrimPrefix(line, "path=")
	}
	return ""
}

// AppendGrepResultSuffix appends grep result stats to a grep title (mode from rawArgs).
func AppendGrepResultSuffix(argsLine string, rawArgs []byte, result string) string {
	mode, err := builtin.ParseGrepOutputMode(parseGrepOutputModeArg(rawArgs))
	if err != nil {
		mode = builtin.GrepOutputFilesWithMatches
	}
	switch mode {
	case builtin.GrepOutputCount:
		n := grepCountResult(result)
		return argsLine + fmt.Sprintf(" · %d matches", n)
	case builtin.GrepOutputContent:
		n := countGrepContentLines(result)
		return argsLine + fmt.Sprintf(" · %d matches", n)
	default:
		n := countGrepPathLines(result)
		return argsLine + fmt.Sprintf(" · %d paths", n)
	}
}

func parseGrepOutputModeArg(rawArgs []byte) string {
	if len(rawArgs) == 0 {
		return ""
	}
	var args map[string]any
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return ""
	}
	if s, ok := args["output_mode"].(string); ok {
		return s
	}
	return ""
}

func grepCountResult(result string) int {
	if m := grepFoundOccurrencesRe.FindStringSubmatch(result); len(m) == 2 {
		n, err := strconv.Atoi(m[1])
		if err == nil && n >= 0 {
			return n
		}
	}
	return 0
}

func countGrepContentLines(result string) int {
	return countGrepBodyLines(result)
}

func countGrepPathLines(result string) int {
	if m := grepFoundFilesRe.FindStringSubmatch(result); len(m) == 2 {
		n, err := strconv.Atoi(m[1])
		if err == nil && n >= 0 {
			return n
		}
	}
	return countGrepBodyLines(result)
}

func countGrepBodyLines(result string) int {
	n := 0
	for _, line := range strings.Split(result, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || isGrepMetaLine(line) {
			continue
		}
		n++
	}
	return n
}

func isGrepMetaLine(line string) bool {
	if strings.HasPrefix(line, "Found ") {
		return true
	}
	if strings.HasPrefix(line, "[Showing results with pagination") {
		return true
	}
	if strings.HasPrefix(line, "（结果已截断") {
		return true
	}
	return strings.HasPrefix(line, "... 已截断")
}

// AppendPathResultSuffix appends path count for glob/list_dir.
func AppendPathResultSuffix(argsLine, result string) string {
	if n := countGrepPathLines(result); strings.HasPrefix(strings.TrimSpace(result), "Found ") {
		return argsLine + fmt.Sprintf(" · %d paths", n)
	}
	n := countNonEmptyLines(result)
	if n > 0 {
		return argsLine + fmt.Sprintf(" · %d paths", n)
	}
	return argsLine
}

func countNonEmptyLines(s string) int {
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}

// UsesHumanDisplay reports tools that use a single-line human title in args.
func UsesHumanDisplay(name string, disp DisplayContext) bool {
	switch name {
	case "read_file", "write_file", "grep", "glob", "list_dir", "agent", "task", "web_fetch":
		return true
	default:
		if disp.IsMCPTool(name) {
			return true
		}
		return isLegacyMCPToolName(name)
	}
}

// IsApplyPatchDisplay reports apply_patch dual-segment title rendering.
func IsApplyPatchDisplay(name string) bool {
	return name == "apply_patch"
}

// HumanToolTitle returns a single-line TUI label when the tool uses a non-function style.
// Empty means use specialized or default rendering.
func HumanToolTitle(name, args, command string, disp DisplayContext) string {
	if IsShellDisplay(name) || IsApplyPatchDisplay(name) {
		return ""
	}
	if UsesHumanDisplay(name, disp) && args != "" {
		return args
	}
	return ""
}

func truncateOneLine(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func formatArgsJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var compact any
	if err := json.Unmarshal(raw, &compact); err != nil {
		s := strings.TrimSpace(string(raw))
		if len(s) > 400 {
			return s[:400] + "..."
		}
		return s
	}
	b, err := json.Marshal(compact)
	if err != nil {
		return string(raw)
	}
	s := string(b)
	if len(s) > 400 {
		return s[:400] + "..."
	}
	return s
}
