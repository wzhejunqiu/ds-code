package logging

import (
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type traceCore struct {
	zapcore.Core
}

func wrapTraceCore(core zapcore.Core) zapcore.Core {
	return &traceCore{Core: core}
}

func (c *traceCore) With(fields []zapcore.Field) zapcore.Core {
	return &traceCore{Core: c.Core.With(fields)}
}

func (c *traceCore) Enabled(lvl zapcore.Level) bool {
	return c.Core.Enabled(lvl)
}

func (c *traceCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(ent.Level) {
		return ce
	}
	return ce.AddCore(ent, c)
}

func (c *traceCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	if ctx := Current(); ctx != nil {
		if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
			fields = append(fields,
				zap.String("trace_id", sc.TraceID().String()),
				zap.String("span_id", sc.SpanID().String()),
			)
		}
	}
	return c.Core.Write(ent, fields)
}
