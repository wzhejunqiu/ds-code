package prompt

import (
	"strings"
	"testing"
	"time"
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
	if !strings.Contains(body, "当前时间：2026-05-23 15:04:05 CST") {
		t.Fatalf("missing time: %q", body)
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
	if strings.Contains(body, "当前时间") {
		t.Fatalf("unexpected time line: %q", body)
	}
	if !strings.Contains(body, "工作区（project_root）：/proj") {
		t.Fatalf("missing project_root: %q", body)
	}
}

func TestMergeSystem_runtimeEnvBeforeAgentsMD(t *testing.T) {
	merged := MergeSystem("BASE", "工作区（project_root）：/x", "AGENTS", "", "", "")
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
