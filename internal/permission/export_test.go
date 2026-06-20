package permission_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permission"
)

func TestIsSensitiveAbs_notExported(t *testing.T) {
	cmd := exec.Command("go", "doc", "github.com/wzhejunqiu/ds-code/internal/permission.IsSensitiveAbs")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("IsSensitiveAbs should not be exported, go doc returned: %s", out)
	}
	outStr := string(out)
	if !strings.Contains(outStr, "no such symbol") &&
		!strings.Contains(outStr, "not found") &&
		!strings.Contains(outStr, "cannot find package") {
		t.Fatalf("unexpected go doc output: %s", out)
	}

	dir := t.TempDir()
	eng := permission.NewEngine("auto", dir, false)
	if eng.SkipSensitiveAbs(dir) {
		t.Fatal("temp workspace root should not be sensitive")
	}
}
