package rgutil

import (
	"path/filepath"
	"strings"
)

// SensitiveExcludeGlobs returns ripgrep --glob negation patterns for sensitive paths.
func SensitiveExcludeGlobs() []string {
	return []string{
		"!**/.env", "!**/.env.*", "!**/.envrc",
		"!**/.netrc", "!**/.npmrc", "!**/.pypirc",
		"!**/id_rsa*", "!**/id_ed25519*", "!**/id_ecdsa*", "!**/id_dsa*",
		"!**/*.pem", "!**/*.key",
		"!**/.ssh/**", "!**/.aws/**", "!**/.docker/**", "!**/.kube/**",
		"!**/.gnupg/**", "!**/credentials/**", "!**/secrets/**",
		"!**/token.json", "!**/secrets.json", "!**/credentials.json",
		"!**/service-account.json", "!**/kubeconfig",
	}
}

// SkipGlobs returns --glob negation patterns for skip_dirs when searching from scopeRoot.
// Explicit scope into a skip dir (scope == skip or under skip/) omits that skip glob.
func SkipGlobs(scopeRoot string, skipDirs []string) []string {
	scope := filepath.ToSlash(strings.Trim(scopeRoot, "/"))
	if scope == "" {
		scope = "."
	}
	var globs []string
	for _, skip := range skipDirs {
		skip = filepath.ToSlash(strings.Trim(skip, "/"))
		if skip == "" {
			continue
		}
		if scope != "." && (scope == skip || strings.HasPrefix(scope, skip+"/")) {
			continue
		}
		globs = append(globs, "!"+skip+"/**")
	}
	return globs
}
