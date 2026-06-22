# v0.1.4 验收标准

> 版本：v0.1.4  
> 状态：规划中  
> 更新日期：2026-06-21  
> 需求：[REQUIREMENTS.md](REQUIREMENTS.md) · 台账：[TOOL_PROMPTS.md](TOOL_PROMPTS.md)

## 1. 总体验收

- [ ] 版本号 v0.1.4
- [ ] `make test` / `make lint` / `make vet` 通过
- [ ] [TOOL_PROMPTS.md](TOOL_PROMPTS.md) 中 **12 项工具 + 共享 Schema** 均为「已定稿」且有你确认记录
- [ ] `CHANGELOG.md` 含 `bash` 改名 breaking 及 `run_in_background` breaking（若本版合入）

**说明**：系统提示词 `prompt.md` 未全部审定时，可不阻塞发布，但须在 CHANGELOG 标明「部分 system 章节待后续」——除非你决定将 system 纳入本版 P0。

## 2. 协作流程验收

- [ ] 无未经你确认的大批量 Desc 替换合入
- [ ] 每个已定稿工具在 PR/对话中有可追溯的确认

## 2.1 FR-0 标准模式验收（每个工具）

| 检查 | 预期 |
|------|------|
| 存在 `<tool>/usage.prompt` | 是 |
| `text.go` 含 `//go:embed usage.prompt` + `RenderDesc()` | 是 |
| `<tool>.go` 的 `Description()` 仅调用 `RenderDesc()` | 是 |
| 无大段 `const Desc*` / `fmt.Sprintf` 拼 Description | 是 |
| `text_test.go`（建议）无 `{{.` 残留 | 是 |
| 参考对照 | 与 [`shell/`](../../internal/tool/builtin/shell/) 结构一致 |

## 3. 逐工具提示词验收

每项：**你已审定文案** + 代码已合入 + 下表「检查」通过。

| 工具 | 检查 | 通过 |
|------|------|------|
| `read_file` | FR-0 + 目录/二进制/分段读取等要点 | [ ] |
| `grep` | FR-0 + pattern/path/output_mode schema | [ ] |
| `glob` | FR-0 + pattern schema | [ ] |
| `list_dir` | FR-0 + path schema | [ ] |
| `diagnostics` | FR-0 + paths/severity schema | [ ] |
| `web_fetch` | FR-0 + url schema | [ ] |
| `web_search` | FR-0 + query schema | [ ] |
| `bash` | FR-0 ✅ + FR-5 行为；schema 含 `run_in_background`、`timeout_ms`；无 `background`/`list_jobs` | [x] |
| `apply_patch` | FR-0 + patch schema | [ ] |
| `write_file` | FR-0 + path/content schema | [ ] |
| `tool_search` | FR-0 + `tool_name` schema | [ ] |
| `agent` | FR-0 + 全部 agent schema | [ ] |
| 共享 `builtin/text.go` | 各 `SchemaPath*` 等定稿 | [ ] |

### AC-3.1 API 抽查（手动，可选）

启动 agent 模式，`-vv` 查看首轮 `tools` JSON：

- 每个内建工具的 `description` 与对应 `usage.prompt` 渲染结果一致
- 无残留 wire 名 `shell`

## 4. `bash` 改名回归

| 检查 | 预期 |
|------|------|
| `tool.NameShell.String()` | `"bash"` |
| `registry` 无 `shell` | 是 |
| `defer_builtin: ["bash"]` | defer 生效 |
| `Engine.Check("bash", …)` | 走 shell 专用逻辑 |
| TUI 工具行 | 显示 `bash` |

**自动化**：`go test ./internal/toolname/... ./internal/permission/... ./internal/tool/...`

## 4.1 bash 工具行为验收（FR-5）

| 检查 | 预期 |
|------|------|
| tools JSON | 含 `run_in_background`、`timeout_ms`；**无** `background`、`list_jobs`、`job_id`、`cancel` |
| `timeout_ms` + 长 sleep（sync 或 bg） | 超时 kill，输出含 deadline exceeded 或 signal killed |
| `timeout_ms` + bg 且已有 stdout | 超时后仍返回已产生 stdout（若有），并附带 `exit:` |
| `run_in_background` | **受** `timeout_ms` / 默认 timeout 限制；阻塞至完成返回 stdout/stderr |
| TUI Running | sync 与 `run_in_background` bash 标题末尾 **递减**倒计时 |
| 退出 ds-code | 本会话 running shell job 被 kill；**无** `/kill` slash |
| [`shell.md`](../../internal/tool/builtin/shell/shell.md) | 与代码、prompt 一致 |

## 5. 非目标（不阻塞发布）

- 历史会话 `shell` tool_call 可 replay
- MCP description 格式变更
- 子代理 `PromptOverlay` 改写
- 系统 `prompt.md` 全部章节审定（除非你升为 P0）

## 6. 建议的手动 smoke

在你定稿的工具改完后任选：

| 场景 | 期望工具 |
|------|----------|
| 「读 README 前 20 行」 | `read_file` |
| 「在 internal/prompt 搜 DefaultSystemBase」 | `grep` |
| 「列出 docs/v0.1.4 下文件」 | `glob` 或 `list_dir`（符合你 Desc 分工） |
| 「跑 prompt 包测试」 | `bash` |
