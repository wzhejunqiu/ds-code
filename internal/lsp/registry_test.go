package lsp

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

func TestServerForExt_go(t *testing.T) {
	reg := BuildRegistry(config.LSPConfig{})
	if got := ServerForExt(reg, ".go"); got != "go" {
		t.Fatalf("ServerForExt(.go) = %q, want go", got)
	}
	// java is registered but disabled until user sets command.
	if got := ServerForExt(reg, ".java"); got != "" {
		t.Fatalf("ServerForExt(.java) = %q, want empty while disabled", got)
	}
	if reg["java"].ID != "java" {
		t.Fatal("java server missing from registry")
	}
}

func TestBuildRegistry_userOverride(t *testing.T) {
	reg := BuildRegistry(config.LSPConfig{
		Servers: map[string]config.LSPServerConfig{
			"java": {Command: "/opt/jdtls", Disabled: false},
		},
	})
	if reg["java"].Command != "/opt/jdtls" {
		t.Fatalf("java command = %q", reg["java"].Command)
	}
	if reg["java"].Disabled {
		t.Fatal("expected java enabled after user command set")
	}
}
