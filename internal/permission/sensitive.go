package permission

import (
	"path/filepath"
	"strings"
)

// sensitiveDirSegments deny a path segment and its subtree (exact name match).
var sensitiveDirSegments = map[string]bool{
	".ssh":        true,
	"credentials": true,
	"secrets":     true,
}

// isSensitiveBasename reports whether a single path segment (file or directory name) is sensitive.
func isSensitiveBasename(name string) bool {
	lower := strings.ToLower(name)
	if lower == ".env" || strings.HasPrefix(lower, ".env.") {
		return true
	}
	if strings.HasPrefix(lower, "id_rsa") || strings.HasPrefix(lower, "id_ed25519") || strings.HasPrefix(lower, "id_ecdsa") {
		return true
	}
	switch lower {
	case "netrc", ".netrc", "token.json", "secrets.json", "credentials.json":
		return true
	}
	if strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key") {
		return true
	}
	return sensitiveDirSegments[lower]
}

// IsSensitiveAbs reports whether abs is on the sensitive denylist (S3).
// Matching is per path segment to avoid false positives (e.g. secrets-management.md).
func IsSensitiveAbs(abs string) bool {
	lower := strings.ToLower(filepath.ToSlash(abs))
	for _, seg := range strings.Split(lower, "/") {
		if seg == "" || seg == "." {
			continue
		}
		if isSensitiveBasename(seg) {
			return true
		}
	}
	return false
}
