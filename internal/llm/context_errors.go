package llm

import "strings"

// IsContextTooLong reports whether err indicates context length exceeded.
func IsContextTooLong(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	phrases := []string{
		"context length exceeded",
		"maximum context length",
		"context window",
		"context is too long",
		"exceeds the context",
		"too many tokens",
		"request too large",
		"payload too large",
		"http 413",
		"status code 413",
		"status: 413",
		"error 413",
	}
	for _, p := range phrases {
		if strings.Contains(s, p) {
			return true
		}
	}
	if strings.Contains(s, "context") && strings.Contains(s, "token") &&
		(strings.Contains(s, "limit") || strings.Contains(s, "exceed")) {
		return true
	}
	return false
}

// IsRateLimit reports whether err is a rate-limit (429) error.
func IsRateLimit(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "rate limit") ||
		strings.Contains(s, "too many requests") ||
		strings.Contains(s, "429")
}

// IsServerError reports whether err is a server-side (5xx) error.
func IsServerError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "500") ||
		strings.Contains(s, "502") ||
		strings.Contains(s, "503") ||
		strings.Contains(s, "504") ||
		strings.Contains(s, "internal server error") ||
		strings.Contains(s, "service unavailable")
}

// IsTransientNetworkError reports whether err is a retryable transient network failure.
func IsTransientNetworkError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	if IsContextTooLong(err) || IsRateLimit(err) {
		return false
	}
	if strings.Contains(s, "invalid request") {
		return false
	}
	phrases := []string{
		"connection reset",
		"connection refused",
		"broken pipe",
		"read: eof",
		"i/o timeout",
		"network timeout",
		"dial tcp",
		"connection closed",
		"no such host",
		"temporary failure",
	}
	for _, p := range phrases {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// IsFinishReasonMaxTokens reports whether err indicates output was truncated
// because max_tokens was reached.
func IsFinishReasonMaxTokens(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "max tokens") ||
		strings.Contains(s, "maximum tokens") ||
		strings.Contains(s, "max_tokens") ||
		strings.Contains(s, "output length")
}
