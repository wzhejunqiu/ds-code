package glob

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

func TestBuildGlobRipgrepArgs(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{Tools: config.ToolsConfig{
		Glob: config.GlobToolConfig{
			RespectGitignore: false,
			IncludeHidden:    true,
		},
		Search: config.SearchToolConfig{SkipDirs: []string{"node_modules"}},
	}}
	tool := &GlobTool{
		Cfg:        cfg,
		Perm:       permission.NewEngine("readonly", dir, false),
		SearchSkip: searchskip.New(nil),
	}

	args := buildGlobRipgrepArgs(tool, "**/*.go", "pkg")
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"--files",
		"--no-ignore",
		"--hidden",
		"--glob",
		"**/*.go",
		"--glob",
		"!.git/**",
		"pkg",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
}
