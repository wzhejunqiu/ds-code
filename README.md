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
- Phase 3+ 的 `CountBreakdown`、`context.truncate_by: tokenizer` 精确截断

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

## 实施阶段

当前：**Phase 0**（脚手架：cobra、`internal/config`、CI）。  
下一步 Phase 1：LLM client、内存 session、基础工具与 `-p` Agent MVP。路线图见 [docs/PLAN.md](docs/PLAN.md)。

## 许可证

待定。
