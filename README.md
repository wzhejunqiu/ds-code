# ds-code

Go 原生 CLI 编码 Agent，对标 Claude Code / Codex 范式：DeepSeek V4、Agent/Plan 双模式、Codex 式 `apply_patch`、项目上下文（`AGENTS.md`、Rules、Skills）与权限沙箱。

**文档**：[PLAN.md](docs/PLAN.md) · [DESIGN.md](docs/DESIGN.md) · [CONFIG.md](docs/CONFIG.md) · [llm-deepseek.md](docs/llm-deepseek.md)

## 要求

- Go 1.26+（见 `go.mod`）
- DeepSeek API Key（调用 LLM 时需要）：`DS_CODE_DEEPSEEK_API_KEY` 或 `DEEPSEEK_API_KEY`

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

   产物：`third_party/tokenizers/libtokenizers.a`（按 OS/arch 本地下载，不入库；见 `.gitignore`）。

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
make test    # 单元测试（缺 lib 时自动 ./scripts/fetch-tokenizers-lib.sh）
make test-tui   # TUI 集成测（-tags=tuitest，见 docs/TUI_INTEGRATION_TEST.md）
make build-tui-test && ./bin/ds-code-tui-test   # 交互式 TUI harness（内置 /tcase 场景）
make test-integration  # LSP 集成测（需 gopls：go install golang.org/x/tools/gopls@latest）
make vet
make lint    # 需本地安装 golangci-lint
make vuln    # govulncheck
```

CI：`.github/workflows/ci.yml`（test、vet、build、golangci-lint、govulncheck）。

## 使用

```bash
export DS_CODE_DEEPSEEK_API_KEY=sk-...

# 非交互单次任务
ds-code -p "在项目根目录找出 main 函数并解释其作用"

# JSON 输出（CI）
ds-code --json -p "列出 internal 目录结构"

# 交互 REPL / TUI（TTY）
ds-code
ds-code -p "解释 @cmd/ds-code/main.go"   # @path / @dir/ 预加载文件或目录

# 会话
ds-code sessions
ds-code resume <session-uuid>

# Plan 模式（只读工具集）
ds-code --plan -p "分析 internal/agent 目录结构并给出重构建议"
```

非 TTY 下 `permission.mode=ask` 时，`shell` 等写操作会被拒绝；脚本请使用 `--permission-mode readonly` 或 `--dangerously-auto`。非 TTY 用 `ds-code -p "..."`，TTY 启动 Bubble Tea 全屏 TUI（`Ctrl+T` 工具日志、`Ctrl+R` reasoning、`Ctrl+L` /context、`Ctrl+C` 取消当前轮次）。

### 交互与上下文

- **`@path` / `@dir/`**：预加载文件/目录（预算见 `context.at_reference_*`）
- **Slash 命令**：`/help`、`/git`、`/mode`、`/effort`、`/thinking`、`/compact`、`/context`、`/clear`、`/resume` 等；`/` 补全支持前缀过滤
- **Git 感知**：启动注入当前分支、默认分支、Git user、`git status`、最近 5 条提交

### 写操作与权限

- **`apply_patch`**：Codex 格式（`*** Begin Patch` …），失败自动回滚
- **`write_file`**：新建或整文件覆盖
- **权限 `ask`**：TTY 下写操作前 `y/N`；非 TTY 拒绝（除非 `auto`）
- **`llm.strict_tools: true`**：工具 schema `additionalProperties: false`，API 走 `/beta`
- **审计**：`audit.enabled: true` 或 `--audit-log` → `~/.ds-code/projects/<id>/audit.jsonl`

### 会话与压缩

- **SQLite**：`~/.ds-code/projects/<project_id>/sessions.db`（按项目分库，0600）
- **自动 compact**：`PrepareRequest` 条件 A/B；API 过长时 compact 后重试（条件 C）
- **手动**：`/compact`、`/context`；`/clear` 换新 session（历史仍保留）

### Checkpoint 与 shell

- **Checkpoint**：`apply_patch` / `write_file` 前自动快照；`/checkpoint list`、`/checkpoint rewind N` 或 `/rewind N`
- **`/btw`**：旁路单次问答，不写 `messages`、默认无 tools（见 `btw.*`）
- **`shell` 后台**：`background=true` 启动；`job_id` 轮询；`cancel` / `list_jobs`

### MCP

在 `~/.ds-code/config/config.yaml` 或项目 `.ds-code/config.yaml` 中配置 `mcp.servers`：

```yaml
mcp:
  servers:
    - name: fs
      command: npx
      args: ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/project"]
```

工具名归一化为 `mcp__{server}__{tool}`；写操作与内置工具同样走 `permission.mode`；单 server 崩溃不影响主进程。

### Rules、Skills、Plan 与其它

- **Rules**：`<git-root>/.ds-code/rules/*.md`
- **Skills**：`.ds-code/skills/<name>/SKILL.md` 或 `~/.ds-code/skills/`；`/skill [name]`
- **Plan**：`ds-code --plan` 或 `/plan` — 只读工具集（+ 可选 `web_fetch`）
- **子代理**：`task` 工具；`/task <prompt>`
- **Web**：`web_fetch` / `web_search`（默认关闭，需 `web.allowlist`）
- **LSP**：`diagnostics` — gopls、typescript-language-server、clangd；Java 自配 jdtls
- **安全**：见 [docs/SECURITY.md](docs/SECURITY.md)

配置示例见 [`configs/example.yaml`](configs/example.yaml)。路线图见 [docs/PLAN.md](docs/PLAN.md)。

## 许可证

[Apache License 2.0](LICENSE)
