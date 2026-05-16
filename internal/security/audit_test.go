package security_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
)

// Spot checks mapping to PLAN.md security audit S1–S14.

func TestS2_pathTraversalDenied(t *testing.T) {
	dir := t.TempDir()
	perm := permission.NewEngine("readonly", dir, false)
	_, err := perm.ResolvePath("../etc/passwd")
	if err == nil {
		t.Fatal("expected denial")
	}
}

func TestS3_sensitiveEnvDenied(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("KEY=x"), 0o600); err != nil {
		t.Fatal(err)
	}
	perm := permission.NewEngine("readonly", dir, false)
	err := perm.Check("read_file", map[string]any{"path": ".env"})
	if err == nil {
		t.Fatal("expected sensitive deny")
	}
}

func TestS10_auditPathUnderProjectData(t *testing.T) {
	root := t.TempDir()
	p := config.DefaultAuditLogPath(root)
	if !strings.Contains(p, "projects") || !strings.HasSuffix(p, "audit.jsonl") {
		t.Fatalf("unexpected audit path: %s", p)
	}
}

func TestS7_checkpointDirUnderProjectData(t *testing.T) {
	root := t.TempDir()
	p := config.DefaultCheckpointDir(root)
	if !strings.Contains(p, "checkpoints") {
		t.Fatalf("unexpected checkpoint dir: %s", p)
	}
}
