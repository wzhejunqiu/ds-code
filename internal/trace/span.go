package trace

import (
	"context"

	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/wzhejunqiu/ds-code"

var tracer = otel.Tracer(instrumentationName)

// Attr keys for span attributes (ds.* namespace).
const (
	AttrSessionID    = "ds.session_id"
	AttrToolName     = "ds.tool.name"
	AttrToolCallID   = "ds.tool.call_id"
	AttrSubRound     = "ds.sub_round"
	AttrLLMModel     = "ds.llm.model"
	AttrSubagentRun  = "ds.subagent.run_id"
	AttrSubagentType = "ds.subagent.type"
)

// Start creates a child span when tracing is enabled; otherwise returns ctx unchanged.
func Start(ctx context.Context, name SpanName, attrs ...attribute.KeyValue) (context.Context, func()) {
	if !Enabled() {
		return ctx, func() {}
	}
	ctx, span := tracer.Start(ctx, name.String(), oteltrace.WithAttributes(attrs...))
	pop := logging.Push(ctx)
	return ctx, func() {
		span.End()
		pop()
	}
}
