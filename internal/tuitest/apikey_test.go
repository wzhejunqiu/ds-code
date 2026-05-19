//go:build tuitest

package tuitest

import (
	"os"
	"testing"
)

func TestEnsureHarnessAPIKey_injectsWhenUnset(t *testing.T) {
	_ = os.Unsetenv("DS_CODE_DEEPSEEK_API_KEY")
	_ = os.Unsetenv("DEEPSEEK_API_KEY")
	key, err := EnsureHarnessAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if key != harnessAPIKey {
		t.Fatalf("key = %q", key)
	}
}
