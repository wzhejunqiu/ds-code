package config

import "github.com/spf13/viper"

// setDefaults registers built-in values before any YAML or env merge.
func setDefaults(v *viper.Viper) {
	v.SetDefault("llm.base_url", "https://api.deepseek.com")
	v.SetDefault("llm.model", "deepseek-v4-pro")
	v.SetDefault("llm.max_tokens", 16384)
	v.SetDefault("llm.timeout", "120s")
	v.SetDefault("llm.strict_tools", false)
	v.SetDefault("llm.thinking.type", "enabled")
	v.SetDefault("llm.reasoning_effort", "max")
	v.SetDefault("llm.subagent.model", "deepseek-v4-flash")
	v.SetDefault("llm.subagent.thinking.type", "disabled")
	v.SetDefault("llm.subagent.reasoning_effort", "high")
	v.SetDefault("llm.subagent.max_turns", 500)

	v.SetDefault("context.window_tokens", 1_048_576)
	v.SetDefault("context.max_output_tokens", 393_216)
	v.SetDefault("context.compact_threshold_ratio", 0.80)
	v.SetDefault("context.collapse_threshold_ratio", 0.85)
	v.SetDefault("context.keep_recent_turns", 6)
	v.SetDefault("context.snip_keep_rounds", 3)
	v.SetDefault("context.truncate_by", "chars")
	v.SetDefault("context.tool_result_max_chars", 100_000)
	v.SetDefault("context.at_reference_max_chars", 128_000)
	v.SetDefault("context.git_snapshot_max_chars", 16_000)
	v.SetDefault("context.at_dir_max_files", 50)
	v.SetDefault("context.at_dir_max_depth", 4)

	v.SetDefault("agent.max_turns", 500)

	v.SetDefault("tools.parallel_tool_calls", true)
	v.SetDefault("tools.defer_mcp", true)
	v.SetDefault("tools.defer_builtin", []string{})
	v.SetDefault("tools.read_file.max_lines", 2000)
	v.SetDefault("tools.read_file.max_bytes", 2<<20)
	v.SetDefault("tools.grep.head_limit", 250)
	v.SetDefault("tools.grep.respect_gitignore", false)
	v.SetDefault("tools.grep.binary", "bundled")
	v.SetDefault("tools.grep.timeout", "20s")
	v.SetDefault("tools.glob.max_results", 100)
	v.SetDefault("tools.glob.respect_gitignore", false)
	v.SetDefault("tools.glob.include_hidden", true)
	v.SetDefault("tools.apply_patch.max_changed_lines", 2000)
	v.SetDefault("tools.shell.timeout", "120s")
	v.SetDefault("tools.shell.max_background", 5)
	v.SetDefault("tools.shell.background_output_max_bytes", 262144)
	v.SetDefault("tools.agent.max_parallel", 3)
	v.SetDefault("tools.agent.summary_max_chars", 16_000)
	v.SetDefault("tools.agent.fork_enabled", true)
	v.SetDefault("tools.agent.auto_background_after", 120)
	v.SetDefault("tools.agent.worktree_ttl", "24h")
	v.SetDefault("tools.agent.worktree_sparse_paths", []string{"/*"})
	v.SetDefault("tools.agent.worktree_symlink_dirs", []string{"node_modules", ".venv", "vendor"})

	v.SetDefault("permission.mode", "ask")

	v.SetDefault("btw.include_recent_turns", 0)
	v.SetDefault("btw.max_tokens", 4096)
	v.SetDefault("btw.count_toward_session", false)

	v.SetDefault("non_interactive.ephemeral_session", true)

	v.SetDefault("audit.enabled", false)

	v.SetDefault("mcp.servers", []any{})

	v.SetDefault("web.fetch_enabled", false)
	v.SetDefault("web.search_enabled", false)
	v.SetDefault("web.allowlist", []string{})

	v.SetDefault("lsp.enabled", true)
	v.SetDefault("lsp.idle_shutdown", "120s")
	v.SetDefault("lsp.diagnostics_timeout", "20s")
	v.SetDefault("lsp.max_files_per_call", 10)
	v.SetDefault("lsp.max_issues_per_file", 20)
	v.SetDefault("lsp.warmup_on_start", []string{})
	v.SetDefault("lsp.servers", map[string]any{})

	v.SetDefault("run_mode", "agent")
	v.SetDefault("tui.copy_on_select", true)
	v.SetDefault("tools.search.skip_dirs", []string{})
}
