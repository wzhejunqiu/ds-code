package permission

import (
	"path/filepath"
	"strings"
)

// Sensitive path substrings (S3). Matched against slash-normalized lowercase paths.
var sensitivePatterns = []string{
	".env",
	".ssh",
	"id_rsa",
	"id_ed25519",
	"credentials",
	"secrets",
}

// IsSensitiveAbs reports whether abs is on the sensitive denylist (S3).
func IsSensitiveAbs(abs string) bool {
	lower := strings.ToLower(filepath.ToSlash(abs))
	for _, p := range sensitivePatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
