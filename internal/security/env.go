package security

import (
	"os"
	"regexp"
	"strings"
)

// IsSecretEnvKey reports whether an environment variable name looks like a secret.
func IsSecretEnvKey(key string) bool {
	u := strings.ToUpper(key)
	if strings.Contains(u, "API_KEY") || strings.Contains(u, "SECRET") ||
		strings.Contains(u, "PASSWORD") || strings.Contains(u, "TOKEN") {
		return true
	}
	return false
}

// IsBlockedEnvKey reports whether key should be excluded from subprocess env.
func IsBlockedEnvKey(key string, patterns []*regexp.Regexp) bool {
	if IsSecretEnvKey(key) {
		return true
	}
	for _, re := range patterns {
		if re != nil && re.MatchString(key) {
			return true
		}
	}
	return false
}

// SafeSubprocessEnv returns parent environment minus blocked keys, plus extra entries.
// extra values override parent keys with the same name.
func SafeSubprocessEnv(extra map[string]string, keyPatterns []*regexp.Regexp) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(os.Environ())+len(extra))
	for _, kv := range os.Environ() {
		i := strings.IndexByte(kv, '=')
		if i <= 0 {
			continue
		}
		key := kv[:i]
		if IsBlockedEnvKey(key, keyPatterns) {
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, kv)
	}
	for k, v := range extra {
		if IsBlockedEnvKey(k, keyPatterns) {
			continue
		}
		out = append(out, k+"="+v)
		seen[k] = true
	}
	return out
}
