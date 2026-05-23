package logging

import (
	"sync/atomic"

	"go.uber.org/zap"
)

var allowSensitive atomic.Bool

// SetAllowSensitiveData configures whether debug logs may include sensitive payloads.
func SetAllowSensitiveData(v bool) {
	allowSensitive.Store(v)
}

// AllowSensitiveData reports whether sensitive debug fields may be logged.
func AllowSensitiveData() bool {
	return allowSensitive.Load()
}

// FieldBody logs the full request body when sensitive logging is allowed, otherwise body_bytes only.
func FieldBody(body []byte) zap.Field {
	if AllowSensitiveData() {
		return zap.String("body", string(body))
	}
	return zap.Int("body_bytes", len(body))
}

// FieldString logs a string field when sensitive logging is allowed; otherwise it is omitted.
func FieldString(key, val string) zap.Field {
	if AllowSensitiveData() {
		return zap.String(key, val)
	}
	return zap.Skip()
}
