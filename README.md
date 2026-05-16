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

## 实施阶段

当前：**Phase 3**（SQLite 会话、compact、resume）。  
路线图见 [docs/PLAN.md](docs/PLAN.md)。

## 许可证

待定。
