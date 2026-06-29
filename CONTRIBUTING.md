# Contributing to ds-code

感谢参与 ds-code 开发与改进。本文档面向 **clone 仓库、从源码构建** 的贡献者；日常使用请参阅 [README.md](README.md)。

## 环境

- Go 1.26+（见 `go.mod`）
- DeepSeek API Key（运行 agent 时）：`DS_CODE_DEEPSEEK_API_KEY` 或 `DEEPSEEK_API_KEY`

## 从源码构建

```bash
git clone https://github.com/wzhejunqiu/ds-code.git
cd ds-code
make build
./bin/ds-code version   # 本地构建显示 dev
```

安装到 `$GOPATH/bin`：

```bash
make install
```

### 版本号

- 源码 / `make build` 默认版本为 `dev`（见 `internal/version/version.go`）
- **正式发布版本**仅由 GitHub Release tag 经 CI ldflags 注入，不在源码中维护版本号
- 发版：GitHub Actions → **Release** workflow → Run workflow，填写 `version`（如 `v0.1.0`）；workflow 先跑 CI，再多平台构建，最后在 `github.sha` 上创建 tag 与 Release。Release 说明自动从 `CHANGELOG.md` 对应版本段提取，无需手填 Description

## Tokenizer 与 CGO

仓库内含 DeepSeek V4 tokenizer 资源，用于 token 计数与上下文截断。

`make build` / `make test` 会自动运行 `./scripts/fetch-tokenizers-lib.sh` 下载 `third_party/tokenizers/libtokenizers.a`（按 OS/arch，不入库）。

## Ripgrep bundled

`grep` / `glob` 的 bundled 后端在编译时 `//go:embed` `internal/tool/builtin/grep/rgbin/rg.tar.gz`（gitignore，不入库）。

`make build` / `make test` / `make vet` / `make verify-release` 等会自动运行 `./scripts/fetch-ripgrep.sh` 下载当前构建机 OS/arch 对应的 ripgrep 15.1.0 包。Linux x86_64 使用 musl 包（与 upstream 发布一致）。

手动拉取：

```bash
make fetch-ripgrep
```

正式发布时 GitHub Release workflow 在各 matrix runner 上分别 fetch 后再 `go build`。

`make verify-charm-v2`（`make check-commit` / git hooks）使用系统 `rg` 命令扫描 import，**需本机 PATH 有 ripgrep**（如 `brew install ripgrep`）；CI 不跑此检查。

### count-tokens 调试

```bash
go run ./cmd/count-tokens -text "hello"
# 精确计数（需 CGO）：
CGO_ENABLED=1 go run -tags cgo ./cmd/count-tokens -text "hello"
```

## 开发命令

```bash
make test              # 单元测试
make test-tui          # TUI 集成测（-tags=tuitest）
make test-integration  # LSP 集成测（需 gopls）
make vet
make lint              # golangci-lint
make staticcheck
make vuln              # govulncheck
make verify-release    # 确认发布二进制不含 tuitest 字符串
make cover             # 覆盖率
```

CI 定义见 `.github/workflows/ci.yml`（test、lint、integration、vuln、musl smoke）。

## Git hooks

克隆后建议安装 pre-commit / pre-push 校验（`core.hooksPath` 为本地配置，不会随 clone 自动生效）：

```bash
make install-hooks
```

| Hook | 命令 | 检查范围 |
|------|------|----------|
| pre-commit | `make check-commit` | 暂存区 `.go` 文件及其 package：gofmt、vet、golangci-lint；全仓 govulncheck |
| pre-push | `make check-push` | 全仓 gofmt、vet、lint、govulncheck（对齐 CI lint / vuln job） |

紧急绕过（不推荐）：`git commit --no-verify` / `git push --no-verify`

## 提交 Pull Request

1. 聚焦改动，保持 diff 小而清晰
2. 确保 `make test` 与 `make lint` 通过
3. 安装 hooks 后 commit/push 会自动跑 `check-commit` / `check-push`
4. **不要**在 PR 中 bump 版本号；Release 由维护者在 GitHub 创建

## 架构文档

- [DESIGN.md](docs/v0.1.0/DESIGN.md) — 模块与流程
- [PLAN.md](docs/v0.1.0/PLAN.md) — 建设路线图与能力矩阵
- [CONFIG.md](docs/v0.1.0/CONFIG.md) — 配置与 CLI
- [llm-deepseek.md](docs/v0.1.0/llm-deepseek.md) — DeepSeek 客户端
- [TUI_INTEGRATION_TEST.md](docs/v0.1.0/TUI_INTEGRATION_TEST.md) — TUI 测试 harness

## 许可证

贡献代码即表示同意 [Apache License 2.0](LICENSE)。
