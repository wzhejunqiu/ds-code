# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- `bash` tool: optional per-call `timeout_ms` (sync and `run_in_background`, max 600000ms); timeout force-kills the subprocess
- `grep` tool: ripgrep 15.1.0 backend（bundled tar.gz embed + `~/.ds-code/bin/rg`）；Claude Code 对齐 Schema（`glob`、`-B/-A/-C`、`head_limit`/`offset` 等）
- `glob` tool: ripgrep 15.1.0 `--files` backend（复用 bundled rg）；输出 `Found N files`；共享 `internal/tool/builtin/rgutil`
- `scripts/fetch-ripgrep.sh` + Makefile `fetch-ripgrep` 目标
- TUI: sync and `run_in_background` `bash` Running title shows a **countdown** via `ToolTimeoutDeadline` + live tick

### Changed

- `grep` 输出格式：`Found N files` / `path:line:text` / `Found X occurrences across Y files`；弃用 `无匹配` 与中文截断行
- `glob` 输出格式：`Found N files` + 相对项目根路径 + 分页脚标；弃用 `无匹配文件` 与 `... 已截断`
- `glob` 后端：`globmatch` Walk → ripgrep `--files`（与 grep 共享 `rgutil`）
- `grep` `head_limit` 三模式通用；默认 **250**（原 200）
- `grep` `path` 与 `glob` 分离（原 `path: pkg/*.go` → `path: pkg` + `glob: *.go`）
- `read_file` tool prompt migrated to `usage.prompt` + `RenderDesc()` (FR-0); LLM parameter **`path` → `filepath`**
- `run_in_background` blocks until job completes (same stdout/stderr format as sync); concurrent batch for multiple bg calls in one turn
- **Removed `/kill` slash command** and TUI kill picker; no manual job management
- **Exit ds-code** now force-kills running shell jobs for the current session (`Manager.Close`)
- **Removed cross-session job recovery** (`loadExisting`); `Open` reconciles stale disk meta only

### Breaking

- `grep` 无匹配：`Found 0 files` / `""` / `Found 0 occurrences across 0 files`（非 `无匹配` / 单个 `0`）
- `glob` 无匹配：`Found 0 files`（非 `无匹配文件`）
- `glob` 显式 `path` 为文件时返回错误（非静默误搜）
- `tools.glob` 新增 `respect_gitignore`、`include_hidden` 配置键
- `grep` `count` 模式受 `head_limit`/`offset` 约束（摘要 X/Y 仍为全量）
- `grep` 正则方言：Go `regexp` → ripgrep / Rust regex
- `tools.grep` 新增 `timeout`、`binary`、`binary_path`、`respect_gitignore` 配置键
- LLM tool name **`shell` → `bash`** (config key `tools.shell` unchanged)
- `read_file` LLM parameter **`path` → `filepath`** (historical tool calls are not aliased)
- Historical tool calls using `background` or `list_jobs` are **not** aliased; use `run_in_background` instead
- LLM **`job_id` / `cancel`** removed from bash schema; use Esc to cancel turn or exit ds-code to kill jobs

## [0.1.3] - 2026-06-20

### Added

- Bubble Tea **v2** stack (`charm.land/bubbletea/v2`, `bubbles/v2`, `lipgloss/v2`, `glamour/v2`)
- Chat area **virtual list** (`LineCatalog`): only visible window is styled/rendered; `plainLines` retained for selection
- TUI double-click word selection and Shift+arrow keyboard selection extension
- Sub-agent `@file` / `@dir/` expansion via `AtExpander` in spawn `ExecuteRun`
- Makefile `verify-charm-v2` guard (no v1 `github.com/charmbracelet/*` imports)

### Changed

- TUI `View()` returns declarative `tea.View` (AltScreen + MouseModeCellMotion); removed HP viewport rendering layer
- Wheel scroll uses `chatScrollY` + virtual catalog instead of viewport HP sync
- Chat virtual list: on-demand styled rendering and incremental `LineCatalog` rebuild (reduces full-transcript refresh cost during scroll/stream)
- Clipboard: prefer `tea.SetClipboard`; fall back to OSC52 / platform copy with toast
- `permission.IsSensitiveAbs` unexported (use `SkipSensitiveAbs` / `ResolveAccessPath` from outside package)

### Fixed

- Restore slash command completion after `/resume` session restore
- Confirm highlighted slash command on Tab (not only move selection)

### Breaking

- **Bubble Tea v1 → v2**: import paths moved to `charm.land/*/v2`; `KeyMsg`/`MouseMsg` replaced by `KeyPressMsg`, typed mouse messages, and `PasteMsg`
- Program options `WithAltScreen` / `WithMouseCellMotion` removed; configured on `tea.View` instead

### Known limitations

- Triple-click line selection and column (block) selection not yet implemented (P2)

## [0.1.2] - 2026-06-20

### Added

- MCP tool results spill to `~/.ds-code/projects/<id>/mcp-result/<session_id>/` with `read_file` hint when truncated
- TUI in-app mouse text selection with copy-on-select (`tui.copy_on_select`, default true)
- TUI smooth wheel scroll with pending drain (native vs integrated terminal profiles; `DS_CODE_SCROLL_SPEED`)
- `tools.search.skip_dirs` for Agent enumeration walk pruning (`.git` always skipped)
- `read_file` rejects non-text files via `textfile.IsTextFile`

### Changed

- Path jail: removed `..` substring pre-filter; `filepath.Clean` + `ensureUnder` only; fixes shell false positives (`git main..branch`, `go test ./...`)
- Path policy unified on `permission.Engine` (`ResolveAccessPath`, `SkipSensitiveAbs`, `CheckWritablePath`)
- User `@file` / `@dir/` references may bypass S3 denylist (explicit user intent; see SECURITY §S3-S)
- MCP tool calls show JSON args preview in TUI and debug logs (`args_preview`)
- Agent search tools no longer follow `.gitignore`; only `.git` + configured `skip_dirs`
- TUI history strips task-notification XML from persisted user messages before display

### Fixed

- iTerm2 wheel scroll leaking SGR mouse escape sequences into the input prompt (fragment reassembly)
- Wheel scroll stuck after copy-on-select (HP disabled only while dragging, not while highlight remains)
- TUI HP viewport sync and `@` reference display alignment

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

| Platform                   | Artifact                   |
| -------------------------- | -------------------------- |
| Linux x86_64 (glibc)       | `ds-code-linux-amd64`      |
| Linux ARM64 (glibc)        | `ds-code-linux-arm64`      |
| Linux x86_64 (Alpine/musl) | `ds-code-linux-musl-amd64` |
| Linux ARM64 (Alpine/musl)  | `ds-code-linux-musl-arm64` |
| macOS Apple Silicon        | `ds-code-darwin-arm64`     |
| macOS Intel                | `ds-code-darwin-x86_64`    |

Windows is not supported in this release.

### Requirements

- **Release binary**: DeepSeek API key (`DS_CODE_DEEPSEEK_API_KEY` or `DEEPSEEK_API_KEY`)
- **Build from source**: Go 1.26+ (see [CONTRIBUTING.md](CONTRIBUTING.md))

### Known limitations

- Session database schema is not migrated automatically; delete `sessions.db` and restart if schema mismatch errors occur
- Non-TTY runs with default `ask` permission reject write operations (use `--permission-mode readonly` or `--dangerously-auto` in scripts)

[0.1.3]: https://github.com/wzhejunqiu/ds-code/releases/tag/v0.1.3
[0.1.2]: https://github.com/wzhejunqiu/ds-code/releases/tag/v0.1.2
[0.1.1]: https://github.com/wzhejunqiu/ds-code/releases/tag/v0.1.1
[0.1.0]: https://github.com/wzhejunqiu/ds-code/releases/tag/v0.1.0
