package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	mu      sync.RWMutex
	current = zap.NewNop()
)

// L returns the process-wide zap logger (nop until Setup runs).
func L() *zap.Logger {
	mu.RLock()
	l := current
	mu.RUnlock()
	return l
}

// Options configures the file log sink.
type Options struct {
	ProjectRoot string
	// Verbosity: 0 = INFO; 1 (-v) = INFO; 2 (-vv) = DEBUG (with caller).
	Verbosity int
	// AllowSensitiveData permits full bodies and paths in debug logs.
	AllowSensitiveData bool
	// TracingEnabled wraps the file core with traceCore for trace_id/span_id injection.
	TracingEnabled bool
}

// TrySetup installs the file logger when possible; on failure it keeps the nop logger.
func TrySetup(opts Options) func() {
	closeLog, err := Setup(opts)
	if err != nil {
		return func() {}
	}
	return closeLog
}

// Setup installs the global logger. Returns a cleanup func that restores the nop logger and closes the file.
func Setup(opts Options) (func(), error) {
	prevSensitive := allowSensitive.Load()
	SetAllowSensitiveData(opts.AllowSensitiveData)
	logDir, err := datadir.EnsureLogsDir(opts.ProjectRoot)
	if err != nil {
		return nil, err
	}
	logPath := filepath.Join(logDir, "ds-code.log")

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("logging: open %s: %w", logPath, err)
	}

	fileLevel := zapcore.InfoLevel
	if opts.Verbosity >= 2 {
		fileLevel = zapcore.DebugLevel
	}

	zapOpts := []zap.Option{zap.AddStacktrace(zapcore.ErrorLevel)}
	if opts.Verbosity >= 2 {
		zapOpts = append(zapOpts, zap.AddCaller())
	}

	logger := zap.New(newCore(zapcore.AddSync(f), fileLevel, opts.TracingEnabled), zapOpts...)
	prev := swap(logger)

	cleanup := func() {
		_ = logger.Sync()
		_ = f.Close()
		swap(prev)
		SetAllowSensitiveData(prevSensitive)
	}

	L().Info("logging initialized",
		zap.String("path", logPath),
		zap.Int("verbosity", opts.Verbosity),
		zap.String("project_id", datadir.ProjectID(opts.ProjectRoot)),
	)
	return cleanup, nil
}

func swap(l *zap.Logger) *zap.Logger {
	mu.Lock()
	defer mu.Unlock()
	prev := current
	if l == nil {
		current = zap.NewNop()
	} else {
		current = l
	}
	return prev
}

// ReplaceForTest swaps the global logger and returns a restore function (tests only).
func ReplaceForTest(l *zap.Logger) func() {
	prev := swap(l)
	return func() { swap(prev) }
}

// NewTestCore returns a zapcore.Core wrapped with traceCore when logctx is active.
func NewTestCore(core zapcore.Core) zapcore.Core {
	if logctxActive {
		return wrapTraceCore(core)
	}
	return core
}

func newCore(ws zapcore.WriteSyncer, level zapcore.Level, tracing bool) zapcore.Core {
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	encoder := zapcore.NewConsoleEncoder(encCfg)
	core := zapcore.NewCore(encoder, ws, level)
	if tracing {
		core = wrapTraceCore(core)
	}
	return core
}
