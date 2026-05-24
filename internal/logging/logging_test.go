package logging_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
)

func TestSetup_writesToProjectLogsDir(t *testing.T) {
	root := t.TempDir()
	cleanup, err := logging.Setup(logging.Options{ProjectRoot: root, Verbosity: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	logging.L().Info("hello from test")
	logPath := config.DefaultLogPath(root)
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "hello from test") {
		t.Fatalf("log file %q missing message: %q", logPath, b)
	}
}

func TestSetup_verbosityLevels(t *testing.T) {
	root := t.TempDir()

	cleanup0, err := logging.Setup(logging.Options{ProjectRoot: root, Verbosity: 0})
	if err != nil {
		t.Fatal(err)
	}
	logging.L().Debug("hidden-debug")
	logging.L().Info("visible-info")
	cleanup0()

	b, err := os.ReadFile(config.DefaultLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	if strings.Contains(body, "hidden-debug") {
		t.Fatal("verbosity 0 should not log DEBUG to file")
	}
	if !strings.Contains(body, "visible-info") {
		t.Fatal("expected INFO in log file")
	}

	cleanup2, err := logging.Setup(logging.Options{ProjectRoot: root, Verbosity: 2})
	if err != nil {
		t.Fatal(err)
	}
	logging.L().Debug("debug-line")
	cleanup2()

	b, err = os.ReadFile(config.DefaultLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "debug-line") {
		t.Fatal("verbosity 2 should log DEBUG to file")
	}
}

func TestDefaultLogPath_underLogs(t *testing.T) {
	root := t.TempDir()
	got := config.DefaultLogPath(root)
	if !strings.HasSuffix(got, filepath.Join("logs", "ds-code.log")) {
		t.Fatalf("path = %q", got)
	}
}

func TestL_beforeSetupIsNop(t *testing.T) {
	prev := zap.NewNop()
	_ = prev
	if logging.L() == nil {
		t.Fatal("expected non-nil logger")
	}
	logging.L().Info("discarded before setup")
}
