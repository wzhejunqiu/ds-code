package trace

import (
	"context"
	"sync/atomic"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

var enabled atomic.Bool

// Enabled reports whether tracing is active.
func Enabled() bool {
	return enabled.Load()
}

// Setup installs the global TracerProvider. Returns a shutdown cleanup.
func Setup(cfg config.TracingConfig) (cleanup func()) {
	enabled.Store(cfg.Enabled)
	logging.SetLogctxActive(cfg.Enabled)

	if !cfg.Enabled {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		return func() {}
	}

	exp, err := newExporter(cfg)
	if err != nil {
		logging.L().Warn("tracing exporter init failed, using noop", zap.Error(err))
		enabled.Store(false)
		logging.SetLogctxActive(false)
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		return func() {}
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(resource.NewWithAttributes(
			"",
			attribute.String("service.name", version.Name),
		)),
	}
	if exp != nil {
		opts = append(opts, sdktrace.WithBatcher(exp))
	}
	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	return func() {
		_ = tp.ForceFlush(context.Background())
		_ = tp.Shutdown(context.Background())
	}
}

func newExporter(cfg config.TracingConfig) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case "":
		return nil, nil
	case "log":
		return newLogExporter(), nil
	case "otlp":
		return otlptracehttp.New(context.Background(),
			otlptracehttp.WithEndpointURL(cfg.OTLPEndpoint),
		)
	default:
		return nil, nil
	}
}
