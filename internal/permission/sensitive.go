package permission

import (
	"path/filepath"
	"strings"
)

// sensitiveDirSegments deny a path segment and its subtree (exact name match).
var sensitiveDirSegments = map[string]bool{
	".ssh":        true,
	".aws":        true,
	".docker":     true,
	".kube":       true,
	".gnupg":      true,
	"credentials": true,
	"secrets":     true,
}

// isSensitiveBasename reports whether a single path segment (file or directory name) is sensitive.
func isSensitiveBasename(name string) bool {
	lower := strings.ToLower(name)
	if lower == ".env" || lower == ".envrc" || strings.HasPrefix(lower, ".env.") {
		return true
	}
	if strings.HasPrefix(lower, "id_rsa") || strings.HasPrefix(lower, "id_ed25519") ||
		strings.HasPrefix(lower, "id_ecdsa") || strings.HasPrefix(lower, "id_dsa") {
		return true
	}
	switch lower {
	case "netrc", ".netrc", ".npmrc", ".pypirc",
		"token.json", "secrets.json", "credentials.json",
		"service-account.json", "kubeconfig":
		return true
	}
	if strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key") {
		return true
	}
	return sensitiveDirSegments[lower]
}

// isSensitiveAbs reports whether abs is on the sensitive denylist (S3).
// Matching is per path segment to avoid false positives (e.g. secrets-management.md).
func isSensitiveAbs(abs string) bool {
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
