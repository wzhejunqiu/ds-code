# v0.1.4 验收标准

> 版本：v0.1.4
> 状态：已发布（`v0.1.4`，2026-06-29）
> 更新日期：2026-06-29
> 需求：[REQUIREMENTS.md](REQUIREMENTS.md) · 台账：[TOOL_PROMPTS.md](TOOL_PROMPTS.md)

## 1. 总体验收

- [x] 版本号 v0.1.4（发布打 `v0.1.4` tag 时由 ldflags 注入）
- [x] `make test` / `make lint` / `make vet` 通过
- [x] [TOOL_PROMPTS.md](TOOL_PROMPTS.md) 中 **8 项** FR-0 工具已定稿；`diagnostics` / `tool_search` / `web_search` **延后**（不阻塞）
- [x] `CHANGELOG.md` 含 `bash` 改名 breaking、`run_in_background` breaking 及 Known limitations

**说明**：系统提示词 `prompt.md` 未全部审定时，不阻塞发布；CHANGELOG 已标明「部分 system 章节待后续」。

## 2. 协作流程验收

- [x] 无未经确认的大批量 Desc 替换合入
- [x] 每个已定稿工具在 PR/对话中有可追溯的确认

## 2.1 FR-0 标准模式验收（已定稿工具）

| 检查                                                    | 预期                                                       |
| ------------------------------------------------------- | ---------------------------------------------------------- |
| 存在 `<tool>/usage.prompt`                              | 是（8 项已定稿工具）                                       |
| `text.go` 含 `//go:embed usage.prompt` + `RenderDesc()` | 是                                                         |
| `<tool>.go` 的 `Description()` 仅调用 `RenderDesc()`    | 是                                                         |
| 无大段 `const Desc*` / `fmt.Sprintf` 拼 Description     | 是（延后 3 项除外）                                        |
| `text_test.go`（建议）无 `{{.` 残留                     | 是                                                         |
| 参考对照                                                | 与 [`shell/`](../../internal/tool/builtin/shell/) 结构一致 |

## 3. 逐工具提示词验收

每项：**你已审定文案** + 代码已合入 + 下表「检查」通过。

| 工具                   | 检查                                                                                         | 通过 |
| ---------------------- | -------------------------------------------------------------------------------------------- | ---- |
| `read_file`            | FR-0 + 目录/二进制/分段读取等要点                                                            | [x]  |
| `grep`                 | FR-0 + FR-6 + §4.2 自动化                                                                    | [x]  |
| `glob`                 | FR-0 + ripgrep + §4.3 自动化；目录列举合并入 glob                                              | [x]  |
| `diagnostics`          | FR-0 + paths/severity schema                                                                 | 延后 |
| `web_fetch`            | FR-0 + url schema                                                                            | [x]  |
| `web_search`           | FR-0 + query schema                                                                          | 延后 |
| `bash`                 | FR-0 + FR-5 行为；schema 含 `run_in_background`、`timeout_ms`；无 `background`/`list_jobs` | [x]  |
| `apply_patch`          | FR-0 + patch schema + read-guard（AC-RG-1～6）                                               | [x]  |
| `write_file`           | FR-0 + path/content schema                                                                   | [x]  |
| `tool_search`          | FR-0 + `tool_name` schema                                                                    | 延后 |
| `agent`                | FR-0 + 全部 agent schema                                                                     | [x]  |
| 共享 `builtin/text.go` | 各 `SchemaPath*` 等沿用现有中文                                                              | [x]  |

### AC-3.1 API 抽查（手动，可选）

启动 agent 模式，`-vv` 查看首轮 `tools` JSON：

- 每个已定稿内建工具的 `description` 与对应 `usage.prompt` 渲染结果一致
- 无残留 wire 名 `shell`

## 4. `bash` 改名回归

| 检查                      | 预期              |
| ------------------------- | ----------------- |
| `tool.NameShell.String()` | `"bash"`          |
| `registry` 无 `shell`     | 是                |
| `defer_builtin: ["bash"]` | defer 生效        |
| `Engine.Check("bash", …)` | 走 shell 专用逻辑 |
| TUI 工具行                | 显示 `bash`       |

**自动化**：`go test ./internal/toolname/... ./internal/permission/... ./internal/tool/...`

## 4.1 bash 工具行为验收（FR-5）

| 检查                                                     | 预期                                                                                       |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| tools JSON                                               | 含 `run_in_background`、`timeout_ms`；**无** `background`、`list_jobs`、`job_id`、`cancel` |
| `timeout_ms` + 长 sleep（sync 或 bg）                    | 超时 kill，输出含 deadline exceeded 或 signal killed                                       |
| `timeout_ms` + bg 且已有 stdout                          | 超时后仍返回已产生 stdout（若有），并附带 `exit:`                                          |
| `run_in_background`                                      | **受** `timeout_ms` / 默认 timeout 限制；阻塞至完成返回 stdout/stderr                      |
| TUI Running                                              | sync 与 `run_in_background` bash 标题末尾 **递减**倒计时                                   |
| 退出 ds-code                                             | 本会话 running shell job 被 kill；**无** `/kill` slash                                     |
| [`shell.md`](../../internal/tool/builtin/shell/shell.md) | 与代码、prompt 一致                                                                        |

## 4.2 grep 工具行为验收（FR-6）

| 检查   | 预期                                                                                 |
| ------ | ------------------------------------------------------------------------------------ |
| 后端   | ripgrep 15.1.0（`bundled` / `system` / `path`）                                      |
| Schema | `glob`、`-B/-A/-C`、`head_limit`/`offset`、`multiline` 等                            |
| 输出   | `Found N files` / `path:line:text` / `Found X occurrences across Y files` + 分页脚标 |
| `.git` | 宽泛 `path` 排除；`path=.git` 空结果                                                 |
| 构建   | `make fetch-ripgrep` + `make test ./internal/tool/builtin/grep/...` 绿               |

**自动化**：`go test ./internal/tool/builtin/grep/...`（含 A1–A20、B1–B27、C、D 组）

## 4.3 glob 工具行为验收（ripgrep 对齐）

| 检查      | 预期                                                             |
| --------- | ---------------------------------------------------------------- |
| 后端      | ripgrep 15.1.0 `--files`（复用 `tools.grep.binary` / `timeout`） |
| Schema    | `pattern`（必填）、`path`（可选目录）                            |
| 输出      | `Found N files` + **相对项目根**路径 + 可选分页脚标              |
| 无匹配    | `Found 0 files`（非 `无匹配文件`）                               |
| 相对路径  | `path: internal/pkg` 时输出 `internal/pkg/a.go`，非 `a.go`       |
| 显式 path | 不存在 → `目录不存在`；文件 → `必须是目录`；UNC 跳过 stat        |
| 默认找全  | `respect_gitignore: false`、`include_hidden: true`               |
| `.git`    | 宽泛 `path` 排除；`path=.git` → `Found 0 files`                  |
| FR-0      | `usage.prompt` + `RenderDesc()` + 中文 Schema                    |

**自动化**：`go test ./internal/tool/builtin/glob/...`（含 G1–G7 输出、H1–H7 输入、G8–G15 I/O、B* 集成、C1 参数快照）

## 5. 非目标（不阻塞发布）

- 历史会话 `shell` tool_call 可 replay
- MCP description 格式变更
- 子代理 `PromptOverlay` 改写
- 系统 `prompt.md` 全部章节审定
- **`diagnostics` / `tool_search` / `web_search` prompt 改写**（延后后续版本）

## 6. 建议的手动 smoke

| 场景                                        | 期望工具                                 |
| ------------------------------------------- | ---------------------------------------- |
| 「读 README 前 20 行」                      | `read_file`                              |
| 「在 internal/prompt 搜 DefaultSystemBase」 | `grep`                                   |
| 「列出 docs/v0.1.4 下文件」                 | `glob`（如 `pattern="*"` 或 `**/*`） |
| 「跑 prompt 包测试」                        | `bash`                                   |

## 3.1 apply_patch read-guard（AC-RG）

| ID      | 场景                                           | 预期                           |
| ------- | ---------------------------------------------- | ------------------------------ |
| AC-RG-1 | 未 read 直接 apply_patch(update A)             | 报错 `ErrMustReadFirstFmt`     |
| AC-RG-2 | sub-round1 read(A)；sub-round2 patch(A)        | 成功                           |
| AC-RG-3 | 同批 tool_calls：read(A)+patch(A)              | 报错 `ErrSameBatchReadEditFmt` |
| AC-RG-4 | patch 仅 Add File                              | 不要求先 read                  |
| AC-RG-5 | 父 session read(A)；子代理 patch(A)（非 Fork） | 失败（不共享已读集合）         |
| AC-RG-6 | Fork seed 含父 read(A)；子 patch(A)            | 成功（水合）                   |
