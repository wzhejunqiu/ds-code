# ds-code

Go 原生 CLI 编码 Agent，对标 Claude Code / Codex 范式：DeepSeek V4、Agent/Plan 双模式、Codex 式 `apply_patch`、项目上下文（`AGENTS.md`、Rules、Skills）与权限沙箱。

**文档**：[PLAN.md](docs/PLAN.md) · [DESIGN.md](docs/DESIGN.md) · [CONFIG.md](docs/CONFIG.md) · [llm-deepseek.md](docs/llm-deepseek.md)

## 要求

- Go 1.26+（见 `go.mod`）
- DeepSeek API Key（Phase 1+ 调用 LLM 时需要）：`DS_CODE_DEEPSEEK_API_KEY` 或 `DEEPSEEK_API_KEY`

## 快速开始

```bash
# 构建 CLI
make build
./bin/ds-code version

# 查看帮助与配置合并（用户级 ~/.ds-code/config/config.yaml + 项目 .ds-code/config.yaml）
./bin/ds-code --help
```

配置示例见 [`configs/example.yaml`](configs/example.yaml)。复制到：

- 用户级：`~/.ds-code/config/config.yaml`
- 项目级：`<git-root>/.ds-code/config.yaml`

## Tokenizer 与可选 CGO

仓库内含 DeepSeek V4 tokenizer 资源（`internal/assets/deepseek-v4/`），用于：

- `cmd/count-tokens` 调试与本地 token 计数
- `CountBreakdown`（compact 条件 A、`/context`）、`context.truncate_by: tokenizer` 精确截断

### 默认（无 CGO）

不启用 CGO 时，相关包使用字符估算或嵌入资源路径，**不**链接 HuggingFace `tokenizers` 静态库。适合 CI 与快速 `go test ./...`。

### 启用 CGO（精确计数 / 工具截断）

1. 下载静态库（按 OS/arch 从 [daulet/tokenizers releases](https://github.com/daulet/tokenizers/releases) 拉取）：

   ```bash
   ./scripts/fetch-tokenizers-lib.sh
   ```

   产物：`third_party/tokenizers/libtokenizers.a`（已在 `.gitignore` 中忽略 tarball）。

2. 带 CGO 构建或测试：

   ```bash
   CGO_ENABLED=1 go build -tags cgo ./cmd/count-tokens
   CGO_ENABLED=1 go test -tags cgo ./internal/tokenizer/deepseek/...
   ```

   `internal/tokenizer/deepseek/cgo.go` 通过 `#cgo LDFLAGS` 链接 `third_party/tokenizers`。

### count-tokens

```bash
go run ./cmd/count-tokens -text "hello"
# 精确计数（需 CGO + fetch 脚本）：
CGO_ENABLED=1 go run -tags cgo ./cmd/count-tokens -text "hello"
```

## 开发

```bash
make test    # 单元测试（默认无 CGO）
make test-tui   # TUI 集成测（-tags=tuitest，见 docs/TUI_INTEGRATION_TEST.md）
make build-tui-test && ./bin/ds-code-tui-test   # 交互式 TUI harness（内置 /tcase 场景）
make test-integration  # LSP 集成测（需 gopls：go install golang.org/x/tools/gopls@latest）
make vet
make lint    # 需本地安装 golangci-lint
make vuln    # govulncheck
```

CI：`.github/workflows/ci.yml`（test、vet、build、golangci-lint、govulncheck）。

## Agent 使用（Phase 1）

```bash
export DS_CODE_DEEPSEEK_API_KEY=sk-...

# 非交互单次任务
ds-code -p "在项目根目录找出 main 函数并解释其作用"

# JSON 输出（CI）
ds-code --json -p "列出 internal 目录结构"

# 交互 REPL（TTY）
ds-code
```

非 TTY 下 `permission.mode=ask` 时，`shell` 等写操作会被拒绝；脚本请使用 `--permission-mode readonly` 或 `--dangerously-auto`。

## Phase 1.5 交互

- **`@path` / `@dir/`**：在用户消息中预加载文件/目录内容（预算见 `context.at_reference_*`）
- **Slash 命令**：行首 `/` 识别；`/help` 列出全部命令；`/git` 刷新 git 快照；`/mode`、`/effort`、`/thinking`、`/clear` 等
- **Git 感知**：启动时自动注入 `git status -sb` + `git diff --stat`（若在 git 仓库内）

```bash
ds-code -p "解释 @cmd/ds-code/main.go"
ds-code    # REPL 中: /help, /git, /mode deepseek-v4-flash
```

## Phase 2 写操作

- **`apply_patch`**：Codex 格式（`*** Begin Patch` …），失败自动回滚
- **`write_file`**：新建或整文件覆盖
- **权限 `ask`**：TTY 下写操作前 `y/N` 确认；非 TTY 仍拒绝（除非 `auto`）
- **`llm.strict_tools: true`**：工具 schema `additionalProperties: false`，API 走 `/beta`
- **审计**：`audit.enabled: true` 或 `--audit-log` → `~/.ds-code/projects/<id>/audit.jsonl`（仅 tool 名 + args 哈希）


## Phase 3 会话与压缩

- **SQLite**：`~/.ds-code/projects/<project_id>/sessions.db`（按项目分库，0600）
- **自动 compact**：`PrepareRequest` 条件 A/B；API 过长时 compact 后重试（条件 C）
- **手动**：`/compact`、`/context`；`/clear` 换新 session（历史仍保留）
- **CLI**：`ds-code sessions`、`ds-code resume <id>`

```bash
ds-code sessions
ds-code resume <session-uuid>
# REPL: /compact, /context, /resume <id>
```

## Phase 4 TUI

交互模式（TTY）启动 **Bubble Tea** 全屏界面：

- **多面板**：对话区、工具日志（`Ctrl+T` 折叠）、输入框、状态栏
- **流式输出**：`reasoning` 默认折叠，`Ctrl+R` 全部展开/收起
- **`/` 补全**：命令列表、前缀过滤、↑↓/Tab 选择
- **`/context`**：累计用量 + 六分项（`Ctrl+L` 或 `/context`）；`--json` 导出
- **状态栏**：模型 · 强度 · 累计 in/out/cache · **费用估算（USD）** · 下次请求预估
- **取消**：运行中 `Ctrl+C` 取消当前轮次

```bash
ds-code          # TUI
ds-code resume <id>
```

非 TTY 仍使用 `ds-code -p "..."`。

## Phase 5 MCP

在 `~/.ds-code/config/config.yaml` 或项目 `.ds-code/config.yaml` 中配置 `mcp.servers`：

```yaml
mcp:
  servers:
    - name: fs
      command: npx
      args: ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/project"]
```

- 工具名归一化为 `mcp__{server}__{tool}`（如 `mcp__fs__read_file`）
- **写操作**（`write_file`、`create_directory` 等）与内置工具一样走 `permission.mode`（`ask` / `readonly` / `auto`）
- 单 server 崩溃不影响主进程（panic 隔离）

## Phase 6 增强

- **Rules**：`<git-root>/.ds-code/rules/*.md` 合并进 system 上下文
- **Skills**：`.ds-code/skills/<name>/SKILL.md` 或 `~/.ds-code/skills/`；`/skill [name]` 激活
- **Plan 模式**：`ds-code --plan` 或 `/plan` — 仅 `read_file` / `grep` / `glob` / `list_dir` / `diagnostics`（+ 可选 `web_fetch`）
- **子代理**：`task` 工具（只读、并发上限 `tools.task.max_parallel`）；`/task <prompt>` 直接派发
- **Web**：`web_fetch` / `web_search`（默认关闭，需 `web.allowlist`）
- **LSP**：`diagnostics` 工具 — 内置 gopls、typescript-language-server、clangd；Java 需在 `lsp.servers.java.command` 自配 jdtls

```bash
ds-code --plan -p "分析 internal/agent 目录结构并给出重构建议"
# REPL: /plan, /agent, /skill my-skill, /task 找出所有 TODO
```

配置见 [`configs/example.yaml`](configs/example.yaml) 中 `web.*` 与 `lsp.*`。

## Phase 7 Checkpoint 与加固

- **Checkpoint**：`apply_patch` / `write_file` 执行前自动快照（`~/.ds-code/projects/<id>/checkpoints/<session>/`）
- **回滚**：`/checkpoint list`、`/checkpoint rewind N` 或 `/rewind N`；历史追加 `role=system` 事件（不进 API）
- **`/btw`**：旁路单次问答，不写 `messages`、默认无 tools（见 `btw.*` 配置）
- **`shell` 后台任务**：`background=true` 启动；`job_id` 轮询输出；`cancel=true` 终止；`list_jobs=true` 列表
- **安全**：威胁模型与 S1–S14 映射见 [docs/SECURITY.md](docs/SECURITY.md)

```bash
# REPL
/checkpoint list
/checkpoint rewind 2
/btw 用一句话解释 compact 条件 A
```

## 实施阶段

当前：**Phase 7**（Checkpoint、/btw、安全文档与审计测试）。
路线图见 [docs/PLAN.md](docs/PLAN.md)。

## 许可证

待定。
