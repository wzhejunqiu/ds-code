package trace_test

import (
	"context"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestSetup_disabled(t *testing.T) {
	cleanup := trace.Setup(config.TracingConfig{Enabled: false})
	defer cleanup()
	if trace.Enabled() {
		t.Fatal("expected disabled")
	}
	ctx, end := trace.Start(context.Background(), trace.SpanRunTurn)
	defer end()
	if oteltrace.SpanFromContext(ctx).SpanContext().IsValid() {
		t.Fatal("expected no span when disabled")
	}
}

func TestSetup_enabledParentChildSameTraceID(t *testing.T) {
	cleanup := trace.Setup(config.TracingConfig{Enabled: true})
	defer cleanup()
	if !trace.Enabled() {
		t.Fatal("expected enabled")
	}

	ctx, endParent := trace.Start(context.Background(), trace.SpanRunTurn)
	defer endParent()
	parentSC := oteltrace.SpanFromContext(ctx).SpanContext()

	ctx, endChild := trace.Start(ctx, trace.SpanLLMChat)
	defer endChild()
	childSC := oteltrace.SpanFromContext(ctx).SpanContext()

	if parentSC.TraceID() != childSC.TraceID() {
		t.Fatalf("trace_id mismatch: %s vs %s", parentSC.TraceID(), childSC.TraceID())
	}
	if parentSC.SpanID() == childSC.SpanID() {
		t.Fatal("expected different span_id")
	}
}

func TestSetup_logExporter(t *testing.T) {
	cleanup := trace.Setup(config.TracingConfig{Enabled: true, Exporter: "log"})
	defer cleanup()
	if !trace.Enabled() {
		t.Fatal("expected enabled")
	}
	_, end := trace.Start(context.Background(), trace.SpanRunTurn)
	end()
}
