# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test

```bash
make build              # → bin/ds-code
make test               # go test -race -count=1 ./...
make test-tui           # TUI 集成测试（-tags=tuitest）
make test-integration   # LSP 集成测试（需 gopls）
make vet                # go vet -copylocks
make lint               # golangci-lint（版本锁定在 .golangci-lint-version）
make vuln               # govulncheck

# 运行单个测试
go test -race -count=1 ./internal/somepkg/... -run TestName

# 构建带 CGO 的精确 tokenizer（需先 ./scripts/fetch-tokenizers-lib.sh）
CGO_ENABLED=1 go build -tags cgo ./cmd/count-tokens
CGO_ENABLED=1 go test -tags cgo ./internal/tokenizer/deepseek/...
```

## Architecture

Go 原生 CLI 编码 Agent，单 Provider **DeepSeek V4**，对标 Claude Code/Codex 范式。Repo module: `github.com/wzhejunqiu/ds-code`（Go 1.26+）。

### 分层

```
cmd/ds-code/           入口（cobra），非交互 -p / 交互 TUI / Plan
  app/                  应用组装：依赖注入（Store、MCP、LSP、Checkpoint、ShellJobs）
internal/agent/        核心 Runner 循环：PrepareRequest → LLM → tool_calls → repeat
  subagent/            子代理 Runner（task 工具调度）
internal/context/      上下文构建：BuildAPIContext、PrepareRequest、CompactAPIContext、CountBreakdown
internal/tool/         Tool 接口 + Registry；内置工具在 builtin/
internal/permission/   权限引擎（readonly/ask/auto + S3 denylist）
internal/session/      Session 领域模型 + Store 接口
  sqlite/              SQLite 持久化（modernc.org/sqlite，按 project_id 分库）
internal/llm/          llm.Client 接口
  deepseek/client/     DeepSeek OpenAI-compatible HTTP 客户端（stream、usage、strict tools /beta）
internal/ui/tui/       Bubble Tea v2 全屏 TUI（`charm.land`；chat/chattool/layout/header 等子包）
internal/lsp/          Language Server 管理器（stdio 子进程，仅 diagnostics）
internal/mcp/          MCP 客户端管理器
internal/checkpoint/   写操作前文件快照
internal/patch/        Codex 式 apply_patch 解析 + 应用
internal/prompt/       系统提示词模板
internal/config/       YAML 配置加载（用户级 ~/.ds-code/config/ + 项目级 .ds-code/）
internal/tokenizer/    DeepSeek V4 tokenizer（CGO 精确 + 纯 Go 字符估算降级）
internal/shelljobs/    后台 shell 任务管理
```

### 核心流程

1. `cmd/ds-code` → 解析配置 → `app.App`（懒初始化依赖）
2. 交互 TUI 走 `ui/tui.Run(agent.Deps)`；非交互走 `agent.Runner.RunTurn` 单轮
3. `Runner.RunTurn` 循环：`context.Service.PrepareRequest` → `llm.Client.Chat` (stream) → `tool.Registry.Execute` → 持久化到 SQLite `messages` 表
4. `/btw` 走 `RunEphemeral`：独立消息切片，不写 DB，默认无 tools
5. Plan 模式（`--plan`）：工具仅 read/grep/glob/list_dir/diagnostics + 可选 web_fetch

### 关键设计约束

- **消息只增**：`messages` 表无 Update/Delete；compact/clear 不删除历史行
- **compact 三层触发**：A) CountBreakdown 超阈值 / B) 累计 prompt_tokens 超比例 / C) API 返回 context-too-long → compact 后重试
- **双层消息模型**：历史记录层（SQLite 全量）+ API 上下文层（内存构建，compact 替换为摘要 + 近 N 轮）
- **权限 S3 denylist 始终生效**：无论 readonly/ask/auto 模式，均拒绝敏感路径的读/写/shell
- **MCP 工具名**：MCP server 原始裸名（如 `semantic_search_nodes`）；写操作与内置工具共享权限检查
- **不实现 edit_file**：编辑统一用 Codex 式 `apply_patch`
- **TUI 取消**：Esc 取消当前轮次（context.Context 贯穿）；Ctrl+C/Ctrl+D 空闲时双击退出

### 项目上下文

- `AGENTS.md`：项目级 agent 指令（自 cwd 向上至 git 根加载）
- `.ds-code/rules/*.md`：项目级 Rules
- `.ds-code/skills/<name>/SKILL.md`：项目级 Skills（用户级 `~/.ds-code/skills/` 也生效）
- `code-review-graph` MCP 工具：知识图谱，优先用 `semantic_search_nodes`/`query_graph`/`detect_changes` 探索代码

### Session 存储

SQLite 文件路径固定：`~/.ds-code/projects/<hex(sha256(project_root))>/sessions.db`（0600 权限）。Schema 不匹配时需删除该文件。
