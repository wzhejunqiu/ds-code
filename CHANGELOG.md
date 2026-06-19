# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.1] - 2026-06-19

### Added

- MCP tools register by **bare name** (original `tool.name`), aligned with AGENTS.md and Cursor
- TUI header **notification zone** for startup notices (MCP skip summary, sensitive-log warning, auto-wrap and scroll)

### Changed

- MCP name conflicts are **skipped with user-visible notices** instead of failing startup
- Sensitive-log warning moved from footer banner to header notification zone
- Project docs reorganized under `docs/v0.1.0/` and `docs/v0.1.1/`

### Breaking

- Removed `mcp__{server}__{tool}` as the registry key; `tool_search` and LLM tool calls use bare names only
- Resuming v0.1.0 sessions: historical `mcp__*` tool calls are display-only; new MCP calls must use bare names

## [0.1.0] - 2026-06-12

First public release of **ds-code** — a Go-native CLI coding agent powered by DeepSeek V4.

### Added

- Interactive TUI and one-shot mode (`ds-code`, `ds-code -p "..."`)
- Agent mode (read/write tools) and Plan mode (`--plan`, read-only analysis)
- Codex-style patch editing, file writes, shell, grep/glob/read_file
- Sub-agents via `task` tool (explore, plan, verification, general-purpose)
- SQLite session persistence, resume, auto/manual context compact
- Checkpoint snapshots and rewind for file changes
- Project context: `AGENTS.md`, Rules (`.ds-code/rules/`), Skills
- `@path` / `@dir/` file references in prompts
- MCP server integration with unified permission checks
- LSP diagnostics (Go, TypeScript, C/C++ via external language servers)
- Optional web fetch/search (off by default)
- Permission modes: readonly, ask (prompt before writes), auto
- Sensitive path denylist and shell safety checks
- Cost estimation in TUI status bar (CNY)
- Slash commands: `/help`, `/compact`, `/context`, `/clear`, `/git`, `/mode`, etc.

### Security

- Baseline audit checklist S1–S14 (see [docs/v0.1.0/SECURITY.md](docs/v0.1.0/SECURITY.md))
- API keys via environment variables only
- Path jail to project root; compact input sanitization

### Platforms

Pre-built binaries (6 `.tar.gz` on GitHub Releases):

| Platform | Artifact |
|----------|----------|
| Linux x86_64 (glibc) | `ds-code-linux-amd64` |
| Linux ARM64 (glibc) | `ds-code-linux-arm64` |
| Linux x86_64 (Alpine/musl) | `ds-code-linux-musl-amd64` |
| Linux ARM64 (Alpine/musl) | `ds-code-linux-musl-arm64` |
| macOS Apple Silicon | `ds-code-darwin-arm64` |
| macOS Intel | `ds-code-darwin-x86_64` |

Windows is not supported in this release.

### Requirements

- **Release binary**: DeepSeek API key (`DS_CODE_DEEPSEEK_API_KEY` or `DEEPSEEK_API_KEY`)
- **Build from source**: Go 1.26+ (see [CONTRIBUTING.md](CONTRIBUTING.md))

### Known limitations

- Session database schema is not migrated automatically; delete `sessions.db` and restart if schema mismatch errors occur
- Non-TTY runs with default `ask` permission reject write operations (use `--permission-mode readonly` or `--dangerously-auto` in scripts)

[0.1.1]: https://github.com/wzhejunqiu/ds-code/releases/tag/v0.1.1
[0.1.0]: https://github.com/wzhejunqiu/ds-code/releases/tag/v0.1.0
