# TUI 集成测试 Harness

本文档说明 `ds-code-tui-test` 与 `internal/tuitest`：在**不调用真实 DeepSeek** 的前提下，用 Mock LLM HTTP Server（SSE 流式）驱动与生产相同的 TUI → agent → `deepseek.Client` 调用链。

## 与单元测试的区别

| | `go test ./...` | TUI harness |
|---|-----------------|-------------|
| LLM | `internal/llm/mock` 或 stub client | 真实 `deepseek.Client` → 本地 mock HTTP |
| TUI | 局部 `model` 测试 | 完整 `turn.RunAsync` + 流式 msg |
| 编译 | 默认 | `-tags=tuitest` |

## 编译隔离

- 发布构建 **`make build`** 不带 `tuitest`，`bin/ds-code` 不含 `/tcase`、mockserver 等。
- Harness 代码均在 `//go:build tuitest` 文件或 `cmd/ds-code-tui-test/` 下。
- 生产包仅增加一行 `trySubmitTUITestSlash` 调用；release 下为空实现。

检查发布二进制：

```bash
make verify-release
```

## 快速开始

无需手动 `export` API Key（harness 会自动注入 `sk-tui-test-mock` 并仍走 `config.LoadAPIKey()`）。

```bash
make build-tui-test
./bin/ds-code-tui-test
```

在 TUI 中：

```
/tcase              # 打开可键盘选择的场景列表
/tcase list         # 同上
/tcase run stream-basic   # 直接运行（仍可手动输入）
```

场景列表：**↑↓** 移动、**Enter** 运行选中项、**Esc** 关闭。

## CI

```bash
make test-tui
```

等价于：

```bash
go test -tags=tuitest -race -count=1 ./internal/tuitest/...
```

## 架构

```
ds-code-tui-test
  → app.RunTUIHarness
  → tui.Run → turn.RunAsync
  → agent.Runner → deepseek.Client
  → POST /chat/completions (mockserver SSE)
```

**Mock 边界**：仅 LLM HTTP。MCP/LSP/Web 在 harness `Config` 中关闭；builtin 工具在临时 `ProjectRoot` 中真实执行。

## API Key

1. `tuitest.EnsureHarnessAPIKey()`：若未设置 `DS_CODE_DEEPSEEK_API_KEY` / `DEEPSEEK_API_KEY`，则写入 `sk-tui-test-mock`。
2. `config.LoadAPIKey()`：与生产相同。
3. `deepseek.Client` 请求头：`Authorization: Bearer <key>`。
4. Mock server **不校验** key，仅可选记录 `LastAuthorization` 供测试断言。

若环境已有 API Key，harness **尊重已有值**。

## 内置场景与剧本

每个场景的 **Prompt、Turn、SSE chunk、tool 参数、期望 UI** 在 **[TUI_TCASE_SCRIPTS.md](./TUI_TCASE_SCRIPTS.md)** 维护（与 `internal/tuitest/scenarios/all.go` 同步）。

| 场景 | 说明 |
|------|------|
| `stream-basic` | 分片 content 流式 |
| `stream-reasoning` | reasoning + content 流式 |
| `tool-read` | `read_file` 两轮 |
| `tool-grep` | `grep` 两轮 |
| `tool-patch` | `apply_patch` 两轮 |
| `tool-multi` | 并行多 tool |
| `tool-shell` | `shell echo` |
| `error-api` | HTTP 400 |
| `error-context` | context 过长后 compact 重试 |
| `md-rich` | Markdown 渲染 |
| `long-stream` | 长流式 chunk |

## 新增场景

1. 在 [`internal/tuitest/scenarios/`](../internal/tuitest/scenarios/) 增加 `*Scenario` 并注册到 `All()`。
2. 在 **[TUI_TCASE_SCRIPTS.md](./TUI_TCASE_SCRIPTS.md)** 补充该场景的剧本说明（Turn、chunk、特殊 registry 行为）。
3. `/tcase` 或 `/tcase list` 打开交互式 Picker；**Enter** 或 `/tcase run <name>` 会 `registry.SetActive(name)` 并提交场景的 `Prompt`。
4. 本地：`make test-tui` 或 `make build-tui-test` 手动验证。

## 目录索引

| 路径 | 作用 |
|------|------|
| `cmd/ds-code-tui-test/` | 交互入口（仅 `tuitest` 构建） |
| `internal/tuitest/mockserver/` | Mock LLM HTTP + SSE |
| `internal/tuitest/scenarios/` | 场景脚本（实现） |
| `docs/TUI_TCASE_SCRIPTS.md` | `/tcase` 场景剧本（文档） |
| `internal/tuitest/stack.go` | 统一组装 app + runner |
| `cmd/ds-code/app/tui_tuitest.go` | `RunTUIHarness` |
| `internal/ui/tui/model/input/submit_hook_*.go` | `/tcase` 挂钩 |

## 故障排查

| 现象 | 处理 |
|------|------|
| `package .../tuitest: build constraints exclude all Go files` | 使用 `-tags=tuitest` |
| `go build ./...` 失败 | 正常；`cmd/ds-code-tui-test` 需 tag，勿在无 tag 时 import `tuitest` |
| Mock 无响应 | 确认 `cfg.LLM.BaseURL` 指向 `mockserver.BaseURL()` |
| tool 场景失败 | 检查临时目录下 `sample.go` 与 patch 路径 |
