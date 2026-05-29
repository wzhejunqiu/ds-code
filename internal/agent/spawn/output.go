package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
)

const (
	// InlineResultMaxBytes is the absolute byte-size cap for inlining result (safety ceiling).
	InlineResultMaxBytes = 1 << 20 // 1 MiB
)

const defaultSummaryMaxChars = 16_000

// summaryMaxChars returns tools.agent.summary_max_chars with a built-in default.
func summaryMaxChars(cfg *config.Config) int {
	if cfg == nil || cfg.Tools.Agent.SummaryMaxChars <= 0 {
		return defaultSummaryMaxChars
	}
	return cfg.Tools.Agent.SummaryMaxChars
}

// DeliveredResult is how a subagent summary is returned to the parent.
type DeliveredResult struct {
	Inline     bool   // true when body is inlined (no spill file)
	Body       string // tool return body or notification inline result
	OutputPath string // set when spilled to disk
}

// ShouldSpillResult reports whether summary must be written to output-file instead of inlined.
func ShouldSpillResult(summary string, maxChars int) bool {
	if maxChars <= 0 {
		maxChars = defaultSummaryMaxChars
	}
	if len(summary) > InlineResultMaxBytes {
		return true
	}
	if utf8.RuneCountInString(summary) > maxChars {
		return true
	}
	return false
}

// DeliverResult decides inline vs spill and optionally writes the output file.
func DeliverResult(dataDir, sessionID, toolCallID, summary, status string, runErr error, cfg *config.Config) DeliveredResult {
	if !ShouldSpillResult(summary, summaryMaxChars(cfg)) {
		return DeliveredResult{Inline: true, Body: summary}
	}
	path := agentOutputPath(dataDir, sessionID, toolCallID)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		logging.L().Warn("cannot create output dir", zap.String("path", path), zap.Error(err))
	} else {
		writeOutputFile(path, summary, status, runErr)
	}
	return DeliveredResult{
		Inline:     false,
		Body:       formatSpillPointer(status, path),
		OutputPath: path,
	}
}

// formatSpillPointer is the short JSON returned to the parent when result is spilled.
func formatSpillPointer(status, outputPath string) string {
	b, err := json.Marshal(map[string]string{
		"status":      status,
		"output_file": outputPath,
	})
	if err != nil {
		return fmt.Sprintf(`{"status":%q,"output_file":%q}`, status, outputPath)
	}
	return string(b)
}

// formatSyncToolReturn builds the sync agent tool return string.
func formatSyncToolReturn(description string, delivered DeliveredResult) string {
	if delivered.Inline {
		if description != "" {
			return fmt.Sprintf("[%s]\n%s", description, delivered.Body)
		}
		return delivered.Body
	}
	return delivered.Body
}

// writeOutputFile persists the full subagent summary to disk.
func writeOutputFile(path, summary, status string, err error) {
	f, fileErr := os.Create(path)
	if fileErr != nil {
		logging.L().Warn("cannot create output file", zap.String("path", path), zap.Error(fileErr))
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "status: %s\n", status)
	if err != nil {
		fmt.Fprintf(f, "error: %s\n", err.Error())
	}
	fmt.Fprintf(f, "\n%s\n", summary)
}

// xmlEscapeText escapes &, <, > for XML text nodes and strips NUL bytes.
func xmlEscapeText(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
