package trace

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogExporter_writesDebugSpan(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	restore := logging.ReplaceForTest(zap.New(core))
	defer restore()

	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(newLogExporter()))
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tr := otel.Tracer("test")
	_, span := tr.Start(context.Background(), "run_turn",
		oteltrace.WithAttributes(attribute.String(AttrSessionID, "sess-1")))
	span.End()

	logs := observed.FilterMessage("span exported").All()
	if len(logs) != 1 {
		t.Fatalf("got %d span exported logs, want 1", len(logs))
	}
	m := logs[0].ContextMap()
	if m["span_name"] != "run_turn" {
		t.Fatalf("span_name = %v", m["span_name"])
	}
	if m["trace_id"] == "" || m["trace_id"] == nil {
		t.Fatalf("missing trace_id: %v", m)
	}
	if m["span_id"] == "" || m["span_id"] == nil {
		t.Fatalf("missing span_id: %v", m)
	}
	attrsRaw := m["attributes"]
	var sessionID string
	switch attrs := attrsRaw.(type) {
	case map[string]interface{}:
		sessionID, _ = attrs[AttrSessionID].(string)
	case map[string]string:
		sessionID = attrs[AttrSessionID]
	default:
		t.Fatalf("attributes = %T %v", attrsRaw, attrsRaw)
	}
	if sessionID != "sess-1" {
		t.Fatalf("session attr = %q", sessionID)
	}
}

func TestLogExporter_writesSpanToFile(t *testing.T) {
	testutil.IsolatedHome(t)
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

	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(newLogExporter()))
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tr := otel.Tracer("test")
	_, span := tr.Start(context.Background(), "run_turn")
	span.End()

	_ = logging.L().Sync()
	data, err := os.ReadFile(datadir.DefaultLogPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "span exported") {
		t.Fatalf("missing span exported log:\n%s", text)
	}
	if !strings.Contains(text, "span_name") {
		t.Fatalf("missing span_name in log:\n%s", text)
	}
}
