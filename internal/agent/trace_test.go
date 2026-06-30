package agent_test

import (
	"context"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
	"github.com/wzhejunqiu/ds-code/internal/trace"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestPermissionDeny_logGetsTraceID(t *testing.T) {
	cleanup := trace.Setup(config.TracingConfig{Enabled: true})
	defer cleanup()

	core, observed := observer.New(zapcore.InfoLevel)
	restore := logging.ReplaceForTest(zap.New(logging.NewTestCore(core)))
	defer restore()

	tr := otel.Tracer("test")
	ctx, span := tr.Start(context.Background(), "tool.read")
	defer span.End()
	pop := logging.Push(ctx)
	defer pop()

	perm := permission.NewEngine(permissionmode.Readonly, "/tmp", false)
	logging.L().Info("tool denied", zap.String("tool", "read"))

	m := observed.All()[0].ContextMap()
	traceID, ok := m["trace_id"].(string)
	if !ok || traceID == "" {
		t.Fatalf("missing trace_id: %v", m)
	}
	spanID, ok := m["span_id"].(string)
	if !ok || spanID == "" {
		t.Fatalf("missing span_id: %v", m)
	}
	_ = perm
	_ = sdktrace.NewTracerProvider()
}

func TestRunConcurrentBatch_logctxBind(t *testing.T) {
	logging.SetLogctxActive(true)
	t.Cleanup(func() { logging.SetLogctxActive(false) })

	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	ctx, end := trace.Start(context.Background(), trace.SpanRunTurn)
	defer end()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer logging.Bind(ctx)()
		if logging.Current() == nil {
			t.Error("expected bound ctx in goroutine")
			return
		}
		_, childEnd := trace.Start(logging.Current(), trace.SpanTool("read"))
		childEnd()
	}()
	<-done
}
