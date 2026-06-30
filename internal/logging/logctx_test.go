package logging_test

import (
	"context"
	"sync"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogctx_pushPop(t *testing.T) {
	logging.SetLogctxActive(true)
	t.Cleanup(func() { logging.SetLogctxActive(false) })

	type ctxKey struct{}
	ctx1 := context.WithValue(context.Background(), ctxKey{}, "a")
	ctx2 := context.WithValue(context.Background(), ctxKey{}, "b")

	pop1 := logging.Push(ctx1)
	if logging.Current() != ctx1 {
		t.Fatal("expected ctx1 on stack")
	}
	pop2 := logging.Push(ctx2)
	if logging.Current() != ctx2 {
		t.Fatal("expected ctx2 on stack")
	}
	pop2()
	if logging.Current() != ctx1 {
		t.Fatal("expected ctx1 after pop2")
	}
	pop1()
	if logging.Current() != nil {
		t.Fatal("expected empty stack")
	}
}

func TestLogctx_bindInGoroutine(t *testing.T) {
	logging.SetLogctxActive(true)
	t.Cleanup(func() { logging.SetLogctxActive(false) })

	type ctxKey struct{}
	parent := context.WithValue(context.Background(), ctxKey{}, "parent")
	pop := logging.Push(parent)
	defer pop()

	var got context.Context
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer logging.Bind(parent)()
		got = logging.Current()
	}()
	wg.Wait()

	if got != parent {
		t.Fatal("goroutine did not inherit parent ctx via Bind")
	}
}

func TestLogctx_disabledNoop(t *testing.T) {
	logging.SetLogctxActive(false)
	pop := logging.Push(context.Background())
	pop()
	if logging.Current() != nil {
		t.Fatal("expected nil when disabled")
	}
}

func initTestTracer(t *testing.T) {
	t.Helper()
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
}

func TestTraceCore_injectsIDs(t *testing.T) {
	logging.SetLogctxActive(true)
	t.Cleanup(func() { logging.SetLogctxActive(false) })
	initTestTracer(t)

	core, observed := observerCore(t)
	logger := zapWithCore(logging.NewTestCore(core))

	tr := otel.Tracer("test")
	ctx, span := tr.Start(context.Background(), "tool.read")
	defer span.End()

	pop := logging.Push(ctx)
	defer pop()

	logger.Info("tool denied")

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("entries = %d", len(entries))
	}
	m := entries[0].ContextMap()
	traceID, ok := m["trace_id"].(string)
	if !ok || traceID == "" {
		t.Fatalf("missing trace_id: %v", m)
	}
	spanID, ok := m["span_id"].(string)
	if !ok || spanID == "" {
		t.Fatalf("missing span_id: %v", m)
	}
}

func TestTraceCore_noSpanNoFields(t *testing.T) {
	logging.SetLogctxActive(true)
	t.Cleanup(func() { logging.SetLogctxActive(false) })

	core, observed := observerCore(t)
	logger := zapWithCore(logging.NewTestCore(core))
	logger.Info("idle")

	m := observed.All()[0].ContextMap()
	if _, ok := m["trace_id"]; ok {
		t.Fatal("unexpected trace_id without span")
	}
}

func observerCore(t *testing.T) (zapcore.Core, *observer.ObservedLogs) {
	t.Helper()
	core, obs := observer.New(zapcore.InfoLevel)
	return core, obs
}

func zapWithCore(core zapcore.Core) *zap.Logger {
	return zap.New(core)
}
