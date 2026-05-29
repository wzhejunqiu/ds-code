package context

import (
	"os"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
)

func TestRecordPromptUsage_detectsDrop(t *testing.T) {
	root := t.TempDir()
	cleanup, err := logging.Setup(logging.Options{ProjectRoot: root, Verbosity: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	s := &Service{promptUsage: newPromptUsage()}
	s.noteStaticHash("sess", "abc")
	s.promptUsage.byID["sess"] = promptUsageEntry{staticHash: "abc", promptTokens: 1000}
	s.RecordPromptUsage("sess", 800)

	logPath := config.DefaultLogPath(root)
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	if !strings.Contains(body, "possible_prompt_cache_hit") {
		t.Fatalf("expected cache hit log, got: %q", body)
	}
	if !strings.Contains(body, "sess") {
		t.Fatal("expected session_id in log")
	}
}

func TestRecordPromptUsage_ignoresWithoutPriorTokens(t *testing.T) {
	s := &Service{promptUsage: newPromptUsage()}
	s.noteStaticHash("sess", "new")
	s.RecordPromptUsage("sess", 500)
	if s.promptUsage.byID["sess"].promptTokens != 500 {
		t.Fatal("expected tokens recorded")
	}
}
