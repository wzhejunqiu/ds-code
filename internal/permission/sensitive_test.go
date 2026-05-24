package permission_test

import (
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permission"
)

func TestIsSensitiveAbs_sensitivePaths(t *testing.T) {
	cases := []string{
		"/proj/.env",
		"/proj/.env.local",
		"/proj/.envrc",
		"/proj/.ssh/config",
		"/proj/.aws/credentials",
		"/proj/.docker/config.json",
		"/proj/.kube/config",
		"/proj/.gnupg/private-keys-v1.d/key",
		"/proj/credentials/api.json",
		"/proj/secrets/vault.txt",
		"/proj/deploy/id_rsa",
		"/proj/cert/server.pem",
		"/proj/token.json",
		"/proj/.npmrc",
		"/proj/service-account.json",
		"/proj/kubeconfig",
	}
	for _, p := range cases {
		if !permission.IsSensitiveAbs(p) {
			t.Errorf("expected sensitive: %q", p)
		}
	}
}

func TestIsSensitiveAbs_allowsBenignPaths(t *testing.T) {
	cases := []string{
		"/proj/docs/secrets-management.md",
		"/proj/internal/credentials_test.go",
		"/proj/pkg/my_secrets_helper.go",
		"/proj/README.md",
	}
	for _, p := range cases {
		if permission.IsSensitiveAbs(p) {
			t.Errorf("expected not sensitive: %q", p)
		}
	}
}

func TestIsSensitiveAbs_workspaceRelative(t *testing.T) {
	root := t.TempDir()
	env := filepath.Join(root, ".env")
	if permission.IsSensitiveAbs(env) != true {
		t.Fatal(".env under workspace should be sensitive")
	}
	ok := filepath.Join(root, "src", "main.go")
	if permission.IsSensitiveAbs(ok) {
		t.Fatal("main.go should not be sensitive")
	}
}
