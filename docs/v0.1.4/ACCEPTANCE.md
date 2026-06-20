# v0.1.4 验收标准

> 版本：v0.1.4  
> 状态：规划中  
> 更新日期：2026-06-21  
> 需求：[REQUIREMENTS.md](REQUIREMENTS.md) · 台账：[TOOL_PROMPTS.md](TOOL_PROMPTS.md)

## 1. 总体验收

- [ ] 版本号 v0.1.4
- [ ] `make test` / `make lint` / `make vet` 通过
- [ ] [TOOL_PROMPTS.md](TOOL_PROMPTS.md) 中 **12 项工具 + 共享 Schema** 均为「已定稿」且有你确认记录
- [ ] `CHANGELOG.md` 含 `bash` 改名 breaking（若本版合入）

**说明**：系统提示词 `prompt.md` 未全部审定时，可不阻塞发布，但须在 CHANGELOG 标明「部分 system 章节待后续」——除非你决定将 system 纳入本版 P0。

## 2. 协作流程验收

- [ ] 无未经你确认的大批量 Desc 替换合入
- [ ] 每个已定稿工具在 PR/对话中有可追溯的确认

## 3. 逐工具提示词验收

每项：**你已审定文案** + 代码已合入 + 下表「检查」通过。

| 工具 | 检查 | 通过 |
|------|------|------|
| `read_file` | `Description()` 非空；含目录/二进制/分段读取等你要求的要点 | [ ] |
| `grep` | Desc + pattern/path/output_mode schema 已更新 | [ ] |
| `glob` | Desc + pattern schema 已更新 | [ ] |
| `list_dir` | Desc + path schema 已更新 | [ ] |
| `diagnostics` | Desc + paths/severity schema 已更新 | [ ] |
| `web_fetch` | Desc + url schema 已更新 | [ ] |
| `web_search` | Desc + query schema 已更新（可与未注册状态一致） | [ ] |
| `bash` | Desc + command/description/background/job schema 已更新；registry 名为 `bash` | [ ] |
| `apply_patch` | Desc + patch schema 已更新 | [ ] |
| `write_file` | Desc + path/content schema 已更新 | [ ] |
| `tool_search` | 文案在 `text.go`；`tool_name` schema 已更新 | [ ] |
| `agent` | Desc + 全部 agent schema 已更新 | [ ] |
| 共享 `builtin/text.go` | 各 `SchemaPath*` 等与你定稿一致 | [ ] |

### AC-3.1 API 抽查（手动，可选）

启动 agent 模式，`-vv` 查看首轮 `tools` JSON：

- 每个内建工具的 `description` 与 `text.go` 定稿一致
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
