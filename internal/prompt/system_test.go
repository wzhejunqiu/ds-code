package prompt

import (
	"strings"
	"testing"
	"time"

	appver "github.com/wzhejunqiu/ds-code/internal/version"
)

func TestFormatRuntimeEnv_pathsAndTime(t *testing.T) {
	now := time.Date(2026, 5, 23, 15, 4, 5, 0, time.FixedZone("CST", 8*3600))
	body := FormatRuntimeEnv("/proj", "/proj/sub", now, []string{"操作系统：darwin 25.5.0 (64 bit)"})
	if !strings.Contains(body, "工作区（project_root）：/proj") {
		t.Fatalf("missing project_root: %q", body)
	}
	if !strings.Contains(body, "当前目录（cwd）：/proj/sub") {
		t.Fatalf("missing cwd: %q", body)
	}
	if !strings.Contains(body, "操作系统：darwin") {
		t.Fatalf("missing platform line: %q", body)
	}
	if !strings.Contains(body, "今日日期：Saturday, 2026-05-23") {
		t.Fatalf("missing date: %q", body)
	}
}

func TestFormatRuntimeEnv_emptyCwdAndZeroTime(t *testing.T) {
	body := FormatRuntimeEnv("/proj", "", time.Time{}, nil)
	if strings.Contains(body, "当前目录") {
		t.Fatalf("unexpected cwd line: %q", body)
	}
	if strings.Contains(body, "今日日期") {
		t.Fatalf("unexpected date line: %q", body)
	}
	if !strings.Contains(body, "工作区（project_root）：/proj") {
		t.Fatalf("missing project_root: %q", body)
	}
}

func TestDefaultSystemBase_injectsBuiltinToolNames(t *testing.T) {
	base := DefaultSystemBase()
	for _, want := range []string{
		"read_file",
		"apply_patch",
		"write_file",
		"glob",
		"grep",
		"bash",
	} {
		if !strings.Contains(base, want) {
			t.Fatalf("missing builtin tool name %q in DefaultSystemBase", want)
		}
	}
	if strings.Contains(base, "使用 Read") || strings.Contains(base, "使用 Edit") {
		t.Fatalf("DefaultSystemBase should not contain generic tool names: %q", base)
	}
	if !strings.Contains(base, "`"+appver.Name+"`") {
		t.Fatalf("DefaultSystemBase should inject app name %q: %q", appver.Name, base)
	}
	if strings.Contains(base, "{{.") {
		t.Fatalf("unexpanded template placeholder in DefaultSystemBase: %q", base)
	}
	if !strings.Contains(base, "`methodName`") || !strings.Contains(base, "`method_name`") {
		t.Fatalf("DefaultSystemBase should preserve markdown inline code from prompt.md: %q", base)
	}
}

func TestMergeSystem_runtimeEnvBeforeAgentsMD(t *testing.T) {
	merged := MergeSystem("BASE", "工作区（project_root）：/x", "AGENTS", "", "", "", "", "")
	if !strings.Contains(merged, "## 运行环境") {
		t.Fatalf("missing runtime section: %q", merged)
	}
	baseIdx := strings.Index(merged, "BASE")
	envIdx := strings.Index(merged, "## 运行环境")
	agentsIdx := strings.Index(merged, "## 项目说明")
	if baseIdx < 0 || envIdx < 0 || agentsIdx < 0 {
		t.Fatalf("sections missing: %q", merged)
	}
	if baseIdx >= envIdx || envIdx >= agentsIdx {
		t.Fatalf("wrong section order: %q", merged)
	}
}
