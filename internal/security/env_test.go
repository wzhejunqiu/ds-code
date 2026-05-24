package security_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/security"
)

func TestSafeSubprocessEnv_excludesSecrets(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "secret")
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("GOPATH", "/tmp/go")
	env := security.SafeSubprocessEnv(nil, nil)
	for _, kv := range env {
		if strings.HasPrefix(kv, "DEEPSEEK_API_KEY=") {
			t.Fatal("API key must not be passed to subprocess")
		}
	}
	foundPath, foundGo := false, false
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			foundPath = true
		}
		if strings.HasPrefix(kv, "GOPATH=") {
			foundGo = true
		}
	}
	if !foundPath {
		t.Fatal("PATH should be in safe env")
	}
	if !foundGo {
		t.Fatal("GOPATH should be passed through")
	}
}

func TestSafeSubprocessEnv_userPattern(t *testing.T) {
	t.Setenv("CUSTOM_CRED", "x")
	t.Setenv("OK_VAR", "y")
	re := regexp.MustCompile(`^CUSTOM_`)
	env := security.SafeSubprocessEnv(nil, []*regexp.Regexp{re})
	for _, kv := range env {
		if strings.HasPrefix(kv, "CUSTOM_CRED=") {
			t.Fatal("pattern should block CUSTOM_CRED")
		}
	}
	found := false
	for _, kv := range env {
		if kv == "OK_VAR=y" {
			found = true
		}
	}
	if !found {
		t.Fatal("OK_VAR should pass through")
	}
}

func TestSafeSubprocessEnv_extra(t *testing.T) {
	env := security.SafeSubprocessEnv(map[string]string{"FOO": "bar"}, nil)
	found := false
	for _, kv := range env {
		if kv == "FOO=bar" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected FOO=bar in env")
	}
}

func TestSafeSubprocessEnv_rejectsSecretExtra(t *testing.T) {
	env := security.SafeSubprocessEnv(map[string]string{"MY_TOKEN": "x"}, nil)
	for _, kv := range env {
		if strings.HasPrefix(kv, "MY_TOKEN=") {
			t.Fatal("extra secret keys must be filtered")
		}
	}
	_ = os.Getenv
}
