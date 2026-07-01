package trace

import (
	"context"

	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

type logExporter struct{}

func newLogExporter() sdktrace.SpanExporter {
	return &logExporter{}
}

func (e *logExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		sc := span.SpanContext()
		fields := []zap.Field{
			zap.String("span_name", span.Name()),
			zap.String("trace_id", sc.TraceID().String()),
			zap.String("span_id", sc.SpanID().String()),
		}
		if parent := span.Parent(); parent.IsValid() {
			fields = append(fields, zap.String("parent_span_id", parent.SpanID().String()))
		}
		if !span.StartTime().IsZero() && !span.EndTime().IsZero() {
			fields = append(fields, zap.Duration("duration", span.EndTime().Sub(span.StartTime())))
		}
		if st := span.Status(); st.Code != codes.Unset {
			fields = append(fields, zap.String("status_code", st.Code.String()))
			if st.Description != "" {
				fields = append(fields, zap.String("status_message", st.Description))
			}
		}
		if attrs := spanAttributes(span); len(attrs) > 0 {
			fields = append(fields, zap.Any("attributes", attrs))
		}
		logging.L().Debug("span exported", fields...)
	}
	return nil
}

func spanAttributes(span sdktrace.ReadOnlySpan) map[string]string {
	attrs := span.Attributes()
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]string, len(attrs))
	for _, kv := range attrs {
		out[string(kv.Key)] = kv.Value.AsString()
	}
	return out
}

func (e *logExporter) Shutdown(context.Context) error {
	return nil
}

var _ sdktrace.SpanExporter = (*logExporter)(nil)
