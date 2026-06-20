# v0.1.3 验收标准

> 版本：v0.1.3  
> 状态：规划中  
> 更新日期：2026-06-20  
> 审核：2026-06-20（五轮）  
> 需求：[REQUIREMENTS.md](REQUIREMENTS.md) · 设计：[DESIGN.md](DESIGN.md)

## 1. 总体验收

- [ ] 版本号标记为 v0.1.3
- [ ] `go.mod` 目标：`charm.land/bubbletea/v2` v2.0.7+、`bubbles/v2` v2.1.0+、`lipgloss/v2` v2.0.4+、`glamour/v2` v2.0.1+（FR-1.1–1.4）
- [ ] **无** `github.com/charmbracelet/{bubbletea,bubbles,lipgloss,glamour}` 在 `go.mod`、`.go` 源码中（注释除外）（FR-1.7、NFR-7）
- [ ] `go.sum` tidy 后无 v1 charm 模块行（FR-1.13）
- [ ] `make test` / `make test-tui` / `make lint` / `make vet` / `make verify-release` 通过
- [ ] `make verify-charm-v2`（或等价脚本）通过（NFR-4）
- [ ] [CHANGELOG.md](../../CHANGELOG.md) v0.1.3 条目（含 **Breaking: Bubble Tea v2**）
- [ ] `permission.IsSensitiveAbs` 不可包外引用（FR-4）

## 2. v2 迁移验收（FR-1）

### AC-2.1 模块路径与影响面

| 检查 | 预期 |
|------|------|
| `rg 'github.com/charmbracelet/(bubbletea\|bubbles\|lipgloss\|glamour)' --glob '*.go' --glob 'go.mod' .` | **无匹配**（或仅注释） |
| `rg 'charm.land/bubbletea/v2' internal/` | TUI 与 tuitest 均 v2 import |
| `rg 'charm.land/lipgloss/v2' internal/logging/warn.go` | v2 lipgloss |
| `go list -m charm.land/bubbletea/v2` | v2.0.7 或更高 |
| charm import `.go` 文件数 | 66 个全部完成路径替换（NFR-6） |

### AC-2.2 声明式 View

| 检查 | 预期 |
|------|------|
| `Model.View()` / `safeModel.View()` 签名 | 返回 `tea.View` |
| `Model.View()` / `safeModel.View()` | **无**副作用（不调用 `applyViewportHP`、不惰性写 `plainLines`）（FR-1.10q） |
| `safe_model` panic fallback | 返回 `tea.View` 且 `AltScreen=true`；**透传** inner `MouseMode`/`Cursor`（FR-1.10k） |
| `run.go` | **无** `WithAltScreen` / `WithMouseCellMotion` |
| `View` 字段 | `AltScreen=true`、`MouseMode=CellMotion` |
| `Init()` | 含 `tea.RequestWindowSize`（FR-1.10n） |
| 运行时 | 全屏 TUI、鼠标滚轮/选区可用 |

### AC-2.3 键鼠消息

| 检查 | 预期 |
|------|------|
| Enter 提交 prompt | 正常 |
| Esc 取消回合 / 关浮层 | 正常 |
| Ctrl+C 双击退出 | 正常；无 alt screen 残留 |
| 空格 / Tab / PgUp/PgDn | `KeyPressMsg.String()` 路径正常（FR-1.10h） |
| 滚轮 | `MouseWheelMsg`；`msg.Mouse().Y` 分区 chat/tool；平滑滚动 |
| 拖拽选区 | `MouseClickMsg` / `MouseReleaseMsg` / `MouseMotionMsg`；`tea.MouseLeft` |
| `KeyReleaseMsg` | `update.go` **不**对 release 重复处理 Enter/Esc/Ctrl+C（FR-1.10aa） |
| 大段粘贴 | `PasteMsg` 正确进入 textinput（FR-1.10o）；bracketed paste 默认启用 |
| Alt+Enter | 多行输入不触发 SubmitLine（FR-1.10ad） |
| `updateInput` Enter | 使用 `KeyPressMsg`/`String()`，**无** `key.Type`/`key.Alt`（FR-1.10h） |
| `bubbles/key.Matches` | 已迁 v2 或改为字符串匹配（`wheel_scroll.go`） |

### AC-2.4 HP 层与 sync 链移除

| 检查 | 预期 |
|------|------|
| `viewport_hp.go` | **已删除** |
| `withHPSync` / `applyViewportHP` / `syncChatViewportHP` / `viewportSyncCmd*` | **无**引用（含 `update.go` 16 处、`ticks.go` 1 处含 `handleNoticeScrollTick`） |
| `toolVP.HighPerformanceRendering` | **无**赋值 |
| `viewportScrollCmdFromLines` / `viewportSyncCmdFor` | **无**引用（`wheel_scroll.go` 已改，FR-1.10t） |
| `viewportHPEnabled` | **无** HP 赋值；浮层/选区禁入逻辑保留为独立函数（FR-1.10s） |
| `viewport.Sync` / `ViewUp` / `ViewDown` | **无**引用 |
| `HighPerformanceRendering` | **无**赋值与测试断言 |
| 长 transcript 滚轮 | 语义不退化（可优于 v0.1.2） |

### AC-2.5 mouse_escape（FR-1.10j / FR-1.10u）

| 检查 | 预期 |
|------|------|
| `mouse_escape_test.go` | 全绿（v2 消息类型；无 `tea.MouseEvent`/`KeyRunes`） |
| `RecoverLeakedMouseKeys` / `AccumulateLeakedMouseKeys` | 输出 v2 鼠标消息；passthrough 用 `KeyPressMsg.Text`；**无** `passthrough.Runes` |
| iTerm2 快速滚轮 | 输入框**无** `[<64;…M` SGR 乱码 |
| `update.go` | 键分支先 `AccumulateLeakedMouseKeys` 再 `updateKey` |

### AC-2.6 scroll/wheel 包（FR-1.10l）

| 检查 | 预期 |
|------|------|
| `scroll/wheel.go` | `ComputeWheelStep(MouseWheelMsg, ...)` |
| `wheel_scroll_test.go` | 使用 v2 滚轮消息构造 |

### AC-2.7 View 纯度（FR-1.10q）

| 检查 | 预期 |
|------|------|
| `rg 'applyViewportHP|plainLines\s*=' internal/ui/tui/model/model.go` 在 `View()` 内 | **无**匹配 |
| 连续滚轮 + 选区 | 无重复 rebuild 抖动；高亮稳定 |

### AC-2.8 build tag 分裂（FR-1.10r）

| 检查 | 预期 |
|------|------|
| `input_release.go` / `input_debug.go` 等 6 文件 | v2 import |
| `make test` + `make test-tui` | 均编译通过 |

### AC-2.9 剪贴板（FR-1.9 / FR-1.10v）

| 检查 | 预期 |
|------|------|
| copy-on-select | 成功 toast；SSH 优先 OSC52（`tea.SetClipboard` 或等价） |
| 复制失败 | footer/status 提示；不 panic |
| 降级 | `internal/ui/clipboard` 仍可用（无 pbcopy 环境） |

### AC-2.10 异步通道与 tuitest harness（FR-1.10x）

| 检查 | 预期 |
|------|------|
| `run.go` `p.Send(msg)` | 迁后仍编译；stream / `TurnDoneMsg` 到达 `Update` |
| `listenPrompt()` + `PromptCh` | permission prompt 经 Cmd 闭包注入；bubble 子代理 ask 可用（FR-1.10ac） |
| `internal/tuitest/harness_test.go` | 全绿；`events` goroutine + `model.Update` 路径可用 v2 消息 |
| Agent 回合 smoke | 流式输出、工具块、`TurnDone` 后 `Running=false` |

## 3. v0.1.2 TUI 能力回归（FR-2）

> 与 [v0.1.2/ACCEPTANCE.md §7–§8](../v0.1.2/ACCEPTANCE.md) **语义**等价。

### AC-3.1 选区与复制

| 步骤 | 预期 |
|------|------|
| 聊天区拖拽 + 松手 | copy-on-select；剪贴板纯文本 |
| `Ctrl+T` 工具面板 | 可选中复制 |
| `tui.copy_on_select: false` | 手动 `Ctrl+Shift+C` / `Cmd+Shift+C` |
| 流式输出中已建立选区 | **可能**错位（FR-7.9 保留） |
| 复制后滚轮 | 高亮保留；聊天区继续滚动（v0.1.2 手动 #42） |
| SSH + OSC52（可选） | `tea.SetClipboard` 或降级提示 |

### AC-3.2 平滑滚动与 profile（FR-2.9）

| 步骤 | 预期 |
|------|------|
| 快速滚轮 | 连续中间帧 |
| PgUp/PgDn | 瞬时；清 pending |
| 滚轮中 PgUp | pending 立即清空 |
| 工具面板滚轮 | 正常 |
| `DS_CODE_SCROLL_SPEED=0.5` / `2` | 步长约为默认一半 / 两倍 |
| Cursor/VS Code 集成终端 | adaptive drain（`scroll.DetectProfile` → integrated） |
| iTerm2 / Terminal.app | proportional drain（`ProfileNative`） |
| `TERM_PROGRAM=cursor` 或 `VSCODE_INJECTED=1` | 与 v0.1.2 相同 profile 分支（FR-2.11） |

### AC-3.3 选区渲染（替代 v0.1.2 HP，AC-8.4）

| 步骤 | 预期 |
|------|------|
| 无选区滚轮 | Cursed Renderer diff；手感不劣于 v0.1.2 |
| 拖拽建立选区 | 高亮正确；可边滚边选 |
| `selDragging` 期间 | 可见窗口 refresh；不依赖 HP；选区 overlay 正确（FR-3.7.6） |

### AC-3.4 iTerm2 SGR（FR-2.10）

| 步骤 | 预期 |
|------|------|
| iTerm2 快速滚轮 | 输入框无 SGR 泄漏乱码 |
| 滚轮后输入字符 | 正常进入 textinput |

### AC-3.5 Markdown / 布局

| 步骤 | 预期 |
|------|------|
| `/tcase run md-rich` | 无 panic；可读 |
| resize | 布局正常 |
| header 通知区滚轮 | 正常 |

### AC-3.6 浮层与滚轮冲突

| 步骤 | 预期 |
|------|------|
| `/help` 打开时聊天区拖拽 | 不建立聊天选区（FR-3.2 首期） |
| `/help` 打开时滚轮 | 忽略聊天滚轮 |
| 关闭浮层后 | 选区/滚轮恢复 |

## 4. 终端矩阵（FR-2.8）

| 环境 | 必测 |
|------|------|
| macOS Terminal / iTerm2 / Ghostty | AC-3.1–3.4 |
| Cursor / VS Code 集成终端 | adaptive drain、复制、AC-3.4 |
| SSH（可选） | 复制降级或 OSC52 |

空闲退出：无 alt screen 残留（FR-3.4.3）。

## 5. 延期项（FR-3 / FR-4）

### AC-5.1 子代理 `@`（FR-3.1，P1）

| 步骤 | 预期 |
|------|------|
| 主会话 `@file` | 仍展开（回归） |
| `task` 子代理 prompt 含 `@file foo.go` | 子代理上下文含 `foo.go` 内容 |
| worktree 子代理 `@` | respect 子代理 `perm.Workspace` |
| 单测 | `internal/agent/spawn/execute_test.go` 新增或扩展 `TestExecuteRun_atExpand`（或等价） |

### AC-5.2 浮层选区（FR-3.2，P1）

| 步骤 | 预期 |
|------|------|
| `/help` 打开 + 聊天区拖拽 | 无聊天选区（最低验收） |
| 权限 prompt 打开 | 同上 |
| 完整版（若实现） | 浮层内可选中复制 |

### AC-5.3 选区增强（FR-3.3，P1–P2）

| 步骤 | 预期 |
|------|------|
| 双击词 | 选中词界（若 P1 实现） |
| `Shift+↓` | 扩展选区并滚动（若 P1 实现） |
| 未实现 | CHANGELOG Known limitations 说明 |

### AC-5.4 `IsSensitiveAbs`（FR-4，P0）

| 检查 | 预期 |
|------|------|
| `rg 'permission\.IsSensitiveAbs' --glob '*.go' .` | 仅 `internal/permission` 包内（或零匹配） |
| `go doc permission.IsSensitiveAbs` | 不存在或 Deprecated 且包外不可引用 |
| `make test` permission 相关 | 全绿 |

### AC-5.5 聊天区虚拟列表（FR-3.7，P1）

| 检查 | 预期 |
|------|------|
| `SyncChat` / `chatVP.SetContent` | **不**再写入全 transcript styled 全文（仅可见窗口 + 可选 overscan） |
| `chat.RenderCache` | block 级缓存 **保留**；`TestRenderCache*` 仍绿 |
| 总行数 / `YOffset` | 与 v0.1.2 滚轮、PgUp/PgDn、跳底、`jumpViewport` 语义一致（FR-3.7.3、FR-3.7.8） |
| resize | 缩放宽高后 catalog/选区坐标正确（FR-3.7.9） |
| 选区 / copy-on-select | `plainLines` 全局坐标正确；AC-3.1 手动项不退化 |
| 流式 tail 更新 | 仅尾 block/catalog invalidate；无明显全屏闪屏 |
| 性能 | `TestLineCatalog_windowCost` 或 benchmark：500 行 vs 1000 行 rebuild **非线性**恶化（FR-3.7.7） |
| 长会话手动 | 100+ 轮 transcript 快速滚轮 + 流式输出：手感不劣于 v0.1.2 |

## 6. 测试清单

### 6.1 须全绿或改写（P0）

- [ ] `internal/ui/tui/chattool/render_test.go`（lipgloss v2）
- [ ] `internal/ui/tui/header_width_test.go`
- [ ] `internal/ui/tui/header/scroll_test.go`
- [ ] `internal/ui/tui/chat/cache_test.go`
- [ ] `internal/ui/tui/chat/planning_test.go`
- [ ] `internal/ui/tui/markdown/incremental_test.go`
- [ ] `internal/ui/tui/markdown/stress_test.go`
- [ ] `internal/ui/tui/selection/selection_test.go`
- [ ] `internal/ui/clipboard/clipboard_test.go`（降级路径；FR-1.9）
- [ ] `internal/ui/tui/scroll/drain_test.go`
- [ ] `internal/ui/tui/header/notice_test.go`（lipgloss v2）
- [ ] `internal/ui/tui/chat/render_test.go`（lipgloss v2）
- [ ] `internal/ui/tui/model/wheel_scroll_test.go`（**移除** `HighPerformanceRendering` 断言）
- [ ] `internal/ui/tui/model/status_test.go`
- [ ] `internal/ui/tui/model/turn/*_test.go`（async、cancel、update、blocks）
- [ ] `internal/ui/tui/model/session/resume_test.go`
- [ ] `internal/ui/tui/model/view/render_test.go`
- [ ] `internal/ui/tui/model/viewport_hp_test.go` → **改写**（`HistoryLoaded`/`SessionResumed` 断言 `scheduleSyncChatView` 或非 HP sync；FR-1.10m）
- [ ] `internal/ui/tui/model/selection_test.go`（**移除** HP 断言）
- [ ] `internal/ui/tui/model/input/mouse_escape_test.go`
- [ ] `internal/ui/tui/model/input_test.go`
- [ ] `internal/ui/tui/model/cancel_test.go`
- [ ] `internal/ui/tui/safe_model_test.go`
- [ ] `internal/ui/tui/component/picker_test.go`
- [ ] `internal/ui/tui/model/subagent/nav_test.go`
- [ ] `internal/ui/tui/markdown/render_test.go`
- [ ] `internal/ui/tui/history/history_test.go`
- [ ] `internal/ui/tui/model/turn/update_test.go`
- [ ] `internal/ui/tui/model/turn/cancel_test.go`
- [ ] `internal/ui/tui/model/turn/blocks_test.go`
- [ ] `internal/ui/tui/model/turn_metrics_test.go`
- [ ] `internal/ui/tui/subagent/registry_test.go`
- [ ] `internal/tuitest/*`

### 6.2 新增建议

- [ ] `TestLineCatalog_totalLines`（FR-3.7.1）
- [ ] `TestSyncChat_visibleWindowOnly`（FR-3.7.2；SetContent 长度 ≈ viewport 行数）
- [ ] `TestLineCatalog_windowCost` / `BenchmarkSyncChatView`（FR-3.7.7）
- [ ] `TestVirtualList_selectionPlainLines`（FR-3.7.4；跨窗口选区复制）
- [ ] `TestVirtualList_streamTailInvalidate`（FR-3.7.5）
- [ ] `TestKeyRelease_ignored`（FR-1.10aa）
- [ ] `TestMouseMotion_dragSelection`（FR-1.10y）
- [ ] `TestRunningMode_chatVPScrollKey`（FR-1.10ab）
- [ ] `TestEventsChannel_turnDone`（FR-1.10x）
- [ ] `TestPasteMsg_textinput`（FR-1.10o）
- [ ] `TestView_noSideEffects`（FR-1.10q）
- [ ] `TestFallbackView_returnsTeaView`（panic 路径）
- [ ] `TestBuildTag_releaseAndTuitestCompile`（FR-1.10r）
- [ ] `TestView_returnsTeaView`（AltScreen/MouseMode）
- [ ] `TestInit_requestsWindowSize`
- [ ] `TestKeyPress_enterEscSpace`（`"enter"` / `"esc"` / `"space"`）
- [ ] `TestMouseWheel_smoothScroll`（v2 `MouseWheelMsg`）
- [ ] `TestMouseEscape_passthroughText`（FR-1.10u；非 `Runes`）
- [ ] `TestWheelScroll_noViewportHP`（FR-1.10t）
- [ ] `TestChatInteractionEnabled_overlayBlocks`（FR-1.10s）
- [ ] `TestDetectProfile_integrated`（FR-2.11；`TERM_PROGRAM=cursor`）
- [ ] `TestTextinput_cursorOrViewCursor`（FR-1.10w；若 v2 需要）
- [ ] `TestSetClipboard_orFallback`
- [ ] `TestScrollSpeed_env`（`DS_CODE_SCROLL_SPEED`）
- [ ] FR-3.1：`TestExecuteRun_atExpand`（`spawn/execute_test.go`）
- [ ] `TestJumpViewport_clampsTotalLines`（FR-3.7.8）
- [ ] `TestSyncChat_gotoBottomVirtualList`（FR-3.7.8）
- [ ] `TestWindowSize_invalidatesCatalog`（FR-3.7.9）
- [ ] `TestAltEnter_noSubmit`（FR-1.10ad）
- [ ] `TestListenPrompt_permissionAsk`（FR-1.10ac）
- [ ] `TestSafeModel_passthroughTeaView`（FR-1.10k）
- [ ] `TestMarkdownRender_colorProfile`（FR-1.10ae）
- [ ] FR-4：`TestIsSensitiveAbs_notExported`

## 7. 手动验证

```bash
make build && make test && make test-tui && make verify-release
# 若已实现：
make verify-charm-v2

bin/ds-code --permission-mode auto

# v2 + v0.1.2 回归（对照 v0.1.2 AC §11 手动项）
# 1–5: 滚轮、PgUp/PgDn、选区复制、工具面板、md-rich
# 6–8: 浮层选区、双击选词、子代理 @（P1 实现后）
# 9: Ctrl+C 双击退出无残留
# 10: iTerm2 快速滚轮无 SGR 乱码（#41）
# 11: 复制后滚轮高亮保留（#42）
# 12: DS_CODE_SCROLL_SPEED=0.5 / 2
# 13: 长 transcript（100+ 轮）滚轮 + 流式 — 虚拟列表不退化（FR-3.7）
```

## 8. 非目标确认

- [ ] **不**保留 v1 bubbletea 双栈
- [ ] Transcript/classic（FR-3.5 P2）可延期
- [ ] Agent/MCP/路径权限 **无**行为变更（除 FR-4）
- [ ] **FR-3.7 虚拟列表** 已实现（原 v0.1.2 FR-9.12「不实现」）
- [ ] Kitty 键盘增强、MCP spill GC、`@` compact 脱敏、**View.OnMouse** **未**实现
- [ ] CI **仍可不**跑 `make test-tui`（NFR-8 本地必跑）

## 9. 发布阻塞

| 项 | 说明 |
|----|------|
| P0 | FR-1 v2 迁移 + FR-2 回归 + FR-4 |
| CHANGELOG | Breaking: charm.land v2 import |
| P1 延期 | FR-3.1–3.3 未合入须在 Known limitations 说明 |
| P1 虚拟列表 | FR-3.7 为 v0.1.3 **In scope**；未合入须在 CHANGELOG 说明 |
| 守卫 | `verify-charm-v2` 纳入 `verify-release` / release workflow（NFR-9） |
| test-tui | 发布前本地 `make test-tui`（CI/release **均未**跑） |
