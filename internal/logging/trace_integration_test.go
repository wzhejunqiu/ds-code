package logging_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/trace"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestSetupThenTraceStart_injectsTraceIDInLogFile(t *testing.T) {
	runFileTraceInjectionTest(t, 2)
}

func TestSetupThenTraceStart_noCaller_injectsTraceIDInLogFile(t *testing.T) {
	runFileTraceInjectionTest(t, 0)
}

func runFileTraceInjectionTest(t *testing.T, verbosity int) {
	t.Helper()
	dir := t.TempDir()
	closeLog, err := logging.Setup(logging.Options{
		ProjectRoot:    dir,
		Verbosity:      verbosity,
		TracingEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer closeLog()
	closeTrace := trace.Setup(config.TracingConfig{Enabled: true})
	defer closeTrace()

	_, end := trace.Start(context.Background(), trace.SpanRunTurn)
	defer end()
	logging.L().Info("user turn start", zap.String("session_id", "test"))

	_ = logging.L().Sync()
	assertLogContains(t, dir, "user turn start", "trace_id")
}

func TestSetupThenObserverLogger_injectsTraceID(t *testing.T) {
	dir := t.TempDir()
	closeLog, err := logging.Setup(logging.Options{
		ProjectRoot:    dir,
		Verbosity:      2,
		TracingEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer closeLog()
	closeTrace := trace.Setup(config.TracingConfig{Enabled: true})
	defer closeTrace()

	tr := otel.Tracer("test")
	ctx, span := tr.Start(context.Background(), "run_turn")
	defer span.End()
	pop := logging.Push(ctx)
	defer pop()

	core, observed := observer.New(zapcore.InfoLevel)
	restore := logging.ReplaceForTest(zap.New(logging.NewTestCore(core)))
	defer restore()
	logging.L().Info("observer probe")

	m := observed.All()[0].ContextMap()
	traceID, ok := m["trace_id"].(string)
	if !ok || traceID == "" {
		t.Fatalf("missing trace_id: %v", m)
	}
	spanID, ok := m["span_id"].(string)
	if !ok || spanID == "" {
		t.Fatalf("missing span_id: %v", m)
	}
}

func assertLogContains(t *testing.T, projectRoot, line, field string) {
	t.Helper()
	data, err := os.ReadFile(datadir.DefaultLogPath(projectRoot))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, line) {
		t.Fatalf("missing log line %q: %s", line, text)
	}
	if !strings.Contains(text, field) {
		t.Fatalf("missing %q in log file:\n%s", field, text)
	}
}
