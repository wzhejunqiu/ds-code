package logging_test

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/logging"
	"go.uber.org/zap/zapcore"
)

func TestFieldBody_redacted(t *testing.T) {
	logging.SetAllowSensitiveData(false)
	f := logging.FieldBody([]byte("secret"))
	if f.Key != "body_bytes" {
		t.Fatalf("key = %q, want body_bytes", f.Key)
	}
}

func TestFieldBody_full(t *testing.T) {
	logging.SetAllowSensitiveData(true)
	f := logging.FieldBody([]byte("secret"))
	if f.Key != "body" || f.String != "secret" {
		t.Fatalf("field = %+v", f)
	}
}

func TestFieldString_skipWhenRedacted(t *testing.T) {
	logging.SetAllowSensitiveData(false)
	f := logging.FieldString("path", "/tmp/x")
	if f.Type != zapcore.SkipType {
		t.Fatalf("expected skip, got %+v", f)
	}
}
