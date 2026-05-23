# tui

ds-code 的交互式终端 UI（Bubble Tea）：聊天记录、输入框、浮层，以及 agent 回合的流式更新。

## 消息流（概览）

```
tea.KeyMsg / tea.WindowSizeMsg / 异步 tea.Msg
        │
        ▼
  model.Update  ──► updateKey（浮层、快捷键）──已处理?──► return
        │
        └──► updateInput（textinput、补全、Enter 提交）
```

`running == true` 时，键盘输入仅用于滚动聊天视口；提交与大部分浮层会禁用，直到本回合结束。

## 浮层（`overlayKind`）

浮层以带边框文本（`overlayText`）渲染在**输入框/页脚下方**，不是独立全屏。类型：

| 种类 | 触发 | 关闭 |
|------|------|------|
| `overlayComplete` | 输入 `/…` 前缀 | Esc；选中项；离开 `/` 前缀 |
| `overlayResume` | `/resume` 带参数，或裸 `/resume` 后拉列表 | Esc；选中会话；改命令 |
| `overlayContext` / `overlayHelp` | `/context`、`?` | Esc / q |
| `overlayPrompt` | 工具权限询问通道 | y / n |

`overlay.Dismiss` 只清理 `state` 中的浮层字段（如 `Complete`、`ResumeSessions`）。complete/resume 的 picker widget 由 `input.HandleCompleteKey`（`PickerKeyCancel`）和 `updateInput`（提交消息时）分别调用 `ClearCompletePicker` / `ClearResumePicker`。

## 输入：斜杠补全 vs 恢复会话列表

每次 textinput 更新后调用 `updateCompletion()`（**含光标闪烁 tick**，与按键同等频率）：

1. 若解析为 `/resume <filter>`，交给 `session.ScheduleResumeFilter` 并返回。
2. 若已离开 resume 模式，调用 `clearResumePicker()`。
3. 否则，若去空格后以 `/` 开头，由 `slash.FilterCommands` **同步**驱动 `overlayComplete` 与 `completePicker`。
4. 非斜杠输入则关闭补全浮层。

**架构差异**：TUI 里只有 `/resume <filter>` 是「随输入 debounce + 异步查 Store + Loading 文案」的浮层；斜杠补全等在内存中过滤，用 `CompleteFilterKey` 早退，不会出现 Loading ↔ 空列表闪烁。

**Enter 分流**（有意拆分）：

- **`updateKey`**：在 `overlayResume` 下处理 Enter（按 `resumeSessions[cursor].ID` 恢复）。无匹配时吞掉 Enter，避免把过滤文本当成会话 ID 提交。
- **`updateInput`**：`overlayResume` 打开时吞掉 Enter（由 picker / updateKey 处理）。
- **`handleCompleteKey`**：当 `completionReadyToSubmit()` 为真时对 Enter 返回 `false`——已输入完整注册命令（如 `/context`）应走 `submitLine`，而非再插入补全项。

`completionReadyToSubmit` 使用 `slash.Parse` + `slash.Lookup`：部分前缀（`/c`）仍停留在补全模式；完整命令或 `/<cmd> …` 参数则提交。

## 恢复会话列表

两种入口：

1. **交互过滤**：`/resume foo` → `listFromStore` 拉取后 `filterSessions` 按 ID 前缀、子串或标题过滤；随输入 debounce（`resumeFilterDebounce`）更新浮层。
2. **裸命令**：`/resume` + Enter → `FetchList` → `resumeListMsg`，最多加载 `resumeListMax`（50）条会话；无会话时显示 `No saved sessions.` 并关闭浮层（与交互过滤不同）。

**已加载哨兵**（`state.ResumeSessions`）：

| 值 | 含义 |
|----|------|
| `nil` | 当前 filter 尚未完成一次拉取（可显示 `Loading sessions…`） |
| 非 `nil`（含 `[]` 空切片） | 已拉取完成；无匹配时 picker 显示 `No matching sessions.`，`ScheduleResumeFilter` 在同 filter 下不再发 Tick |

`ApplyResumeSessions` 将 `filterSessions` 的 `nil` 规范为 `[]`，以便与「未加载」区分。filter 变化时会先把 `ResumeSessions` 置 `nil` 再显示 Loading。

`ScheduleResumeFilter` 在过滤串未变且 `ResumeSessions != nil` 时提前返回（光标闪烁、重复 `UpdateCompletion` 不会重复查库），避免重置 `Cursor`。过滤变化时 `ResetSelection`；列表变短时用 `ClampSelection` 保留合法索引。

`session.PageSize()` 按终端高度估算可见行数（约 `height/5`，限制在 4–14），使列表能放在输入框下方。`syncAllViews` 在 `OverlayResume` 且 `ResumeSessions != nil` 时同步 picker（空列表也会刷新 `OverlayText`）。

`LazyStore.ListSessions` 仅返回已落库会话（`AppendMessage` 后）；未发消息的 pending session 不会出现在 `/resume` 列表。列表 title 经 `session.OneLine` 压单行；首条消息后异步 LLM 标题见 `internal/agent/subagent/title.go`。

## 布局

`view.Layout`（[`model/view/render.go`](model/view/render.go)）按**内容行数**计算聊天/工具视口高度，并受终端高度上限约束，使输入框紧贴 transcript 下方，而非钉在屏幕底部；由 `SyncChat` / `SyncTool` 在同步视口内容时调用。

## 主题

共享色板在 `internal/ui/theme`（`colors.go`）。TUI 样式在 `styles.go`；picker 列表样式在 `component/styles.go`。

## Agent 回合桥接（`run.go`）

用户提交（`model_input.go` 的 `submitLine`）会追加 user + 空 assistant 块，设置 `running = true`，再 `go runTurnAsync(...)`。

```
submitLine ──► goroutine runTurnAsync
                    │
                    ├─► turnStartedMsg{cancel}
                    ├─► streamContentMsg / streamReasoningMsg / tool* / planning*
                    └─► turnDoneMsg{result, err}
                         │
                         ▼
              events chan ──► tea.Program.Send ──► model.Update
```

`Deps.Events` 为带缓冲通道（64）。回调使用**非阻塞**发送，避免 UI 卡住 agent；极高吞吐时可能丢弃个别 delta。

Esc：`turnStartedMsg` 到达后 `requestCancelTurn` 调用 `cancel()`；否则设置 `turnEscPending` 并立即显示中断行。中断后的流式消息由 `turnEventsAllowed()` 忽略。

## 流式聊天 transcript

| `chatBlock` 角色 | 来源 |
|------------------|------|
| `chatRoleUser` | `submitLine` |
| `chatRoleAssistant` | `streamContentMsg`、`streamReasoningMsg`；工具开始、分段结束、回合结束时 finalize |
| `chatRoleTool` | `toolStartMsg` / `toolEndMsg` |
| `chatRolePlanning` | `submitLine` 提交时；`planningStartMsg` / `planningEndMsg` 在后续 LLM 子轮次（round>0）与流式输出间隙 |
| `chatRoleInterrupt` | Esc 取消 |

`ensureStreamingAssistant`（`model_turn.go`）决定追加到最后一个 assistant 块还是新建一块。工具行会**切断**分段，多轮子轮次呈现为：assistant → tool → assistant → …

`applyTurnMetrics` 将 `TurnResult.TurnDuration` 与 reasoning 时长挂到**最后一条有可见内容**的 assistant 块（避免工具轮后留下空 assistant 块导致时长显示错位）。

`thinkingTickMsg` 用于刷新 live 的「thinking …」/ planning 文案；超过 10s 后 tick 间隔由 100ms 变为 1s（`chat.go` 中 `thinkingFineDuration`）。

Agent 循环细节见 `internal/agent/README.md`。

## 会话历史（`history.go`）

持久化聊天在 `session.Store`（`~/.ds-code/projects/<project_id>/sessions.db`）。TUI 另维护内存中的 `[]chatBlock` 用于渲染。

```
session.Message[]  ──chatBlocksFromMessages──►  []chatBlock  ──renderChat──► viewport
         ▲                                              │
         │                                              │
   agent.RunTurn 追加                             loadSessionChat
```

**何时加载历史**

| 场景 | 命令 / 处理 |
|------|-------------|
| TUI 启动 | `Init` → `loadInitialHistory` → `historyLoadedMsg` |
| `/resume <id>` | `resumeSession` → `sessionResumedMsg`（替换 `m.chat`） |

**Message → block 映射**

- **User**：每条消息对应一个 `chatRoleUser` 块。
- **Assistant**：有 content/reasoning 时 → `chatRoleAssistant`；`ToolCallsJSON` 展开为多个 `chatRoleTool`（参数来自 `tool.DisplaySummary`，结果通过 `findToolMessage` 匹配后续 tool 行）。
- **Tool**：不单独渲染；在展开 assistant 的 tool_calls 时消费。
- **System**：仅当内容为 `interruptSessionMarker()` 时 → `chatRoleInterrupt`（Esc 取消会写入会话，便于 `/resume` 恢复）。

加载时 `reasoningOpen` 跟随 `m.reasoningAll`（Ctrl+R），恢复的历史与全局 reasoning 展开状态一致。

`ReasoningDurationMS`、`TurnDurationMS` 会复制到 assistant 块，用于「thought」/「task took」等页脚文案。

## 工具行展示约定

TUI 工具块标题由 `internal/tool.DisplaySummary` 生成人类可读单行（非 `tool_name (json)`）：

| 工具 | 示例 |
|------|------|
| `read_file` | `Read sample.go L1-3` |
| `write_file` | `Write sample.go` |
| `grep` / `glob` / `list_dir` | `Grepped … in ds-code` / `Searched files …` / `List tuitest` |
| `shell` | `{description}` + 浅色命令名片段（`cd, 2+`）；展开显示完整 `command` |
| `apply_patch` | 每文件一行 `Edit file.go` + 绿色 `+N` + 红色 `-N` |
| `task` / `web_fetch` | `Task: …` / `Fetch https://…` |
| `mcp__*` | `MCP server · tool` |

`grep`/`glob`/`list_dir` 结束时可能追加 `· N matches` / `· N paths`。持久化会话中的 `ToolCallsJSON` 仍为原始 JSON；仅展示层变化。

## 不可复制类型（传参与遍历）

部分依赖在按值复制后会破坏堆/GC 语义（`-race` 下可能表现为 `found bad pointer in Go heap`，不一定有 `DATA RACE` 报告）。TUI 代码应遵守：

| 类型 | 约定 |
|------|------|
| `state.State`（含 `sync.WaitGroup`） | 仅 `*state.State` 传递，禁止按值复制 |
| `textinput.Model` / `viewport.Model` | 可放在 `model.Model` 内按 bubbles 惯例持有；**跨函数**传 `*textinput.Model` / `*viewport.Model`（见 `model/view.Render`） |
| `lipgloss.Style` | 包级变量可直接 `.Render`；跨函数或循环内用 `*lipgloss.Style`，勿 `s2 := style` 后在多行渲染中复用 |
| `strings.Builder` | 仅作函数内局部变量；**不要**放入会被 `range` 或赋值复制的 struct（`chat.Block` 使用 `string`） |
| `chat.Block` | 可按值存在于 slice 中；遍历需读内容时用 `for i := range blocks { b := &blocks[i] }` |

本地检查：`make vet`（`go vet -copylocks ./...`）。

配置路径与 YAML 键见 `internal/config/README.md` 与 `docs/CONFIG.md`。
