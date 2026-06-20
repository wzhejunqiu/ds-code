# ds-code v0.1.3 版本文档

> 版本：v0.1.3  
> 状态：已发布（`v0.1.3`，2026-06-20）  
> 基线版本：v0.1.2  
> 更新日期：2026-06-20  
> 审核：2026-06-20（五轮；对照 v0.1.2 基线代码 **66** 个 charm import 文件 + [UPGRADE_GUIDE_V2](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md)）

## 概述

v0.1.3 聚焦三类工作：

1. **Charm 终端 UI 栈一次性迁 v2**（需求 1）：将 Bubble Tea 生态从 v1（`github.com/charmbracelet/*`）**直接**升级到 v2（`charm.land/*/v2`），含 `bubbletea`、`bubbles`、`lipgloss`、`glamour` 及关联 `x/ansi` 等；**跳过** v1.3.x 中间态，避免双次迁移。
2. **迁移后全量回归**（需求 2）：`make test` / `make test-tui` / 手动 TUI 清单 / 关键终端矩阵，确保 v0.1.2 已交付能力（选区复制、平滑滚轮、Markdown、header 通知区等）**语义不退化**（实现路径可因 Cursed Renderer 变化）。
3. **历史延期项补全 + 长 transcript 性能**（需求 3）：v0.1.0–v0.1.2 标记延期的条目；**新增** v0.1.2 曾明确不做的 **React 式虚拟列表**（FR-3.7 / 原 FR-9.12），与删 HP 后的 v2 Cursed Renderer 配合，避免极长会话全量 `SetContent`。

## 文档索引

| 文档 | 说明 |
|------|------|
| [REQUIREMENTS.md](REQUIREMENTS.md) | 功能与非功能需求、用户故事、范围边界 |
| [DESIGN.md](DESIGN.md) | v2 依赖矩阵、迁移清单、TUI 重写面、延期项设计 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | 验收标准、回归清单、手动验证步骤 |

## 背景与动机

### 为何直接 v2，不经过 v1.3

| 考量 | 说明 |
|------|------|
| **双次迁移成本** | v1.2.4 → v1.3.10 → v2 需改两遍 TUI；ds-code 约 50+ TUI 文件，一次性迁 v2 更省总工时 |
| **生态方向** | 2026-02 起 Charm 官方主推 v2（Cursed Renderer、声明式 `tea.View`）；`bubbles`/`lipgloss`/`glamour` 均已发布 `charm.land/*/v2` |
| **长期收益** | mode 2026/2027 Unicode、内置 colorprofile、原生 OSC52 剪贴板、SSH 渲染性能 — 与 ds-code 长 transcript Agent TUI 场景契合 |
| **生产验证** | Crush 等已长期跑 v2 栈 |

当前 `go.mod`（v0.1.2）与 v0.1.3 目标：

| 模块 | v0.1.2 | v0.1.3 目标（v2 生态） |
|------|--------|------------------------|
| bubbletea | `github.com/.../bubbletea` v1.2.4 | `charm.land/bubbletea/v2` **v2.0.7+** |
| bubbles | v0.20.0 | `charm.land/bubbles/v2` **v2.1.0+** |
| lipgloss | v1.0.0 | `charm.land/lipgloss/v2` **v2.0.4+** |
| glamour | v0.8.0 | `charm.land/glamour/v2` **v2.0.1+** |
| x/ansi | v0.11.6 | **v0.11.7+** |

> **注意**：v2 模块路径统一为 `charm.land/...`；`github.com/charmbracelet/bubbles/v2` 等 **不可**作为 require path（module 声明与 path 不一致）。

### v0.1.2 遗留延期项

[v0.1.2/ACCEPTANCE.md §12](../v0.1.2/ACCEPTANCE.md#12-非目标确认) 中多项延期纳入 v0.1.3；v2 迁移阶段可顺带：

- 用 `tea.SetClipboard` / `tea.ReadClipboard` **评估**替代部分 `internal/ui/clipboard` 外部命令路径（SSH 友好）
- 声明式 `View.AltScreen` / `View.MouseMode` 替代 `WithAltScreen` / `WithMouseCellMotion`

## 变更摘要

### 需求 1：Bubble Tea v2 迁移

| 领域 | v0.1.2 | v0.1.3（目标） |
|------|--------|----------------|
| import 路径 | `github.com/charmbracelet/bubbletea` | `charm.land/bubbletea/v2` |
| `View()` | `string` | `tea.View`（声明式 AltScreen / MouseMode / Cursor） |
| 键盘 | `tea.KeyMsg` | `tea.KeyPressMsg` / `KeyReleaseMsg`；空格为 `"space"` |
| 鼠标 | `tea.MouseMsg`（struct；`msg.X`/`Action`/`MouseButton*`） | `MouseClickMsg` 等；坐标 `msg.Mouse().X/Y`；`MouseLeft` 等常量 |
| 粘贴 | `KeyMsg` + `Paste`/`Runes` | `tea.PasteMsg`（`Content`）；`updateInput` 须显式分支 |
| 终端特性 | `WithAltScreen()`、`EnterAltScreen` 等 Cmd | `View.AltScreen`、`View.MouseMode` 字段 |
| 滚动渲染 | HP + `SyncScrollArea` + `withHPSync` | **Cursed Renderer** + **虚拟列表**（FR-3.7）；删 `viewport_hp.go` |
| 滚轮输入 | `tea.MouseMsg` + `MouseButtonWheel*` | `MouseWheelMsg`；`scroll/wheel.go` 同步改签名（FR-1.10l） |
| 键位匹配 | `tea.KeyType` + `bubbles/key.Matches` | `KeyPressMsg.String()` + bubbles v2 `key` 或集中键位表（FR-1.10h） |
| iTerm2 SGR | `mouse_escape.go` 从 `KeyRunes` 恢复 `MouseMsg` | 适配 v2 键鼠类型；迁后须 iTerm2 回归（FR-1.10j） |
| 剪贴板 | `pbcopy` / OSC 52 自研 | 优先 `tea.SetClipboard`；保留 `internal/ui/clipboard` 降级 |

### 需求 2：回归验证

与先前规划相同：`make test`、`make test-tui`、v0.1.2 TUI AC §7–§8 语义等价。

### 需求 3：延期项（摘要）

| ID | 优先级 | 说明 |
|----|--------|------|
| §5.4 删除 `IsSensitiveAbs` | **P0** | v0.1.2 承诺 |
| FR-7.8 浮层选区 | **P1** | `/help`、权限 prompt |
| FR-7.10–11 选区增强 | **P1** | 双击/键盘扩展 |
| FR-3.1 子代理 `@` | **P1** | spawn AtExpander |
| **FR-3.7 虚拟列表** | **P1** | 原 FR-9.12；仅渲染可见窗口 |
| FR-3.5 transcript/classic | **P2** | 可选；v2 声明式 View 更易预埋 |

## 审核结论（2026-06-20，五轮）

对照 v0.1.2 代码库与 Bubble Tea v2 Upgrade Guide，文档缺口与处置如下（已同步至 REQUIREMENTS / DESIGN / ACCEPTANCE）。

### 第一轮（结构 / HP / 键位骨架）

| 缺口 | 严重度 | 处置 |
|------|--------|------|
| 键位 API 仅列 `KeyPressMsg`，未覆盖 `tea.KeyType` / `key.Matches` / 全仓库 `func(tea.KeyMsg)` 签名 | P0 | FR-1.10h；DESIGN §3.2 |
| `input/mouse_escape.go`（iTerm2 SGR 泄漏恢复）未列入迁移面 | P0 | FR-1.10j；AC-3.4 |
| `scroll/wheel.go` 仍用 `MouseButtonWheel*`，与 `wheel_scroll.go` 割裂 | P0 | FR-1.10l |
| `withHPSync` / `applyViewportHP` 调用链与 HP 删除策略未写清 | P0 | FR-1.10m；DESIGN §3.4 |
| 影响范围遗漏 `internal/logging/warn.go`、`internal/ui/theme` | P0 | REQUIREMENTS §5 |
| ACCEPTANCE §5 延期项为占位 | P1 | ACCEPTANCE §5 展开 |
| `DS_CODE_SCROLL_SPEED` / `DetectProfile` 误当作可延期 | P1 | FR-2.9 |
| AC 仍引用 HP 渲染 | P1 | ACCEPTANCE AC-3.2 / AC-3.3 |
| 测试清单不完整 | P1 | ACCEPTANCE §6 |
| NFR-4 无具体脚本；CI 未跑 `test-tui` | P1 | DESIGN §10；NFR-8 |
| `go.sum` v1 残留、`muesli/termenv` 直依 | P2 | FR-1.13；DESIGN §2.4 |
| glamour 子包路径未列 | P2 | FR-1.10i |
| `Init()` 未请求 `RequestWindowSize` | P2 | FR-1.10n；DESIGN §3.1 |

### 第二轮（Upgrade Guide + 代码细读）

| 缺口 | 严重度 | 处置 |
|------|--------|------|
| **`Model.View()` 含副作用**（`applyViewportHP()`、惰性 `plainLines` 重建）与 v2 声明式 `tea.View` 冲突 | P0 | FR-1.10q；DESIGN §3.5 |
| **`PasteMsg` 未列入**：v2 粘贴不再走 `KeyMsg.Paste`/`Runes`；`updateInput` default 分支会误吞 | P0 | FR-1.10o；AC-2.3 |
| **鼠标常量 / 坐标 API 未列全**：`MouseButtonLeft`→`MouseLeft`、`MouseEvent(msg)`→`msg.Mouse()`、`handleMouseWheel` 的 `msg.Y` | P0 | FR-1.10p；DESIGN §2.3 |
| **`mouse_escape` 仍依赖 `KeyRunes`/`passthrough.Runes`**，与 `KeyPressMsg.Text` 不兼容 | P0 | 扩展 FR-1.10j |
| **`updateInput` Enter 判定仍用 `key.Type`/`key.Alt`**（`update.go` ~L238） | P0 | FR-1.10h |
| **`handleMouse(tea.MouseMsg)` 统一入口 + `MouseAction*` 分支**须拆为 v2 分类型或 interface 分发 | P0 | FR-1.10d |
| **v1 守卫仅 grep `bubbletea`**，未覆盖 `bubbles`/`lipgloss`/`glamour` 旧 path | P0 | FR-1.13；DESIGN §10 |
| **`safe_model.fallbackView()` 仍返回 `string`**，panic 路径须 `tea.View` + `AltScreen` | P1 | FR-1.10k；AC-2.2 |
| **build tag 分裂文件**（`*_release.go` / `*_tuitest.go` / `input_debug.go`）共 6 个仍 import v1 | P1 | FR-1.10r；DESIGN §4 |
| **FR-4 未列 `permission/path_api.go` 内部 `IsSensitiveAbs` 调用** | P1 | FR-4.1 |
| **`internal/ui/clipboard` 未写入 In scope**（FR-1.9 有描述无边界） | P1 | REQUIREMENTS §5 |
| **测试清单仍缺** `status_test`、`turn/*_test`、`session/resume_test`、`view/render_test` | P1 | ACCEPTANCE §6 |
| **`markdown/styles.go` 缓存 `termenv.Profile` + `lipgloss.ColorProfile()`** v2 对齐未写 | P1 | FR-1.10i |
| **`RequestWindowSize` v2 语义**（返回 `Msg` 非 `Cmd`）与 Init `Batch` 组合 | P2 | DESIGN §3.1 |
| **tuitest 可用 `tea.WithWindowSize`** 做确定性尺寸 | P2 | ACCEPTANCE §6.2 |
| **`tea.WithoutCatchPanics()`** 迁后须确认 v2 仍支持 | P2 | DESIGN §3.1 |

### 第三轮（代码耦合细读 + HP 删除链 + 剪贴板/CI）

| 缺口 | 严重度 | 处置 |
|------|--------|------|
| **`viewportHPEnabled()` 双重语义**（HP 开关 + 浮层/权限 prompt 禁滚轮选区）；删 HP 时不可误删禁入逻辑 | P0 | FR-1.10s；DESIGN §3.4 |
| **`wheel_scroll.go` 仍调 `viewportScrollCmdFromLines`/`viewportSyncCmdFor`**（定义在将删的 `viewport_hp.go`） | P0 | FR-1.10t |
| **`syncChatViewportHP`/`withHPSync` 散布 `update.go` 16 处 + `ticks.go` 1 处** | P0 | 扩展 FR-1.10m；AC-2.4 |
| **`selection_update` 拖拽开始调 `applyViewportHP()`** | P0 | FR-1.10q / FR-1.10m |
| **`mouse_escape` 全链路**：`[]tea.MouseMsg` 构造、`passthrough.Runes`、`parseLeakedSGRButton→MouseEvent` | P0 | FR-1.10u |
| **`asyncCopy` goroutine + `clipboard.Write`** 与 `tea.SetClipboard` Cmd 语义未设计 | P1 | FR-1.10v；DESIGN §3.7 |
| **bubbles textinput v2 光标 / `View.Cursor`** 未评估 | P1 | FR-1.10w |
| **HP 相关单测断言**（`wheel_scroll_test`/`selection_test`/`viewport_hp_test`） | P1 | ACCEPTANCE §6 |
| **`scroll.DetectProfile()` 无单测**；v0.1.2 AC §12 曾误标为可延期（代码已实现） | P1 | FR-2.11 |
| **测试清单缺** `header/notice_test.go`、`chat/render_test.go` | P2 | ACCEPTANCE §6 |
| **`verify-charm-v2` 仅文档建议**；`Makefile`/`release.yml` 尚无 target | P1 | NFR-9；DESIGN §10 |
| **CI + release 均不跑 `test-tui`** | P1 | NFR-8（扩展说明） |
| **v0.1.2 FR-7.14**（不改 AltScreen 用户行为）与 v2 声明式迁移的说明 | P2 | 已知限制 |

### 第四轮（异步通道 / Init 语义 / 拖拽 / tuitest / KeyRelease）

| 缺口 | 严重度 | 处置 |
|------|--------|------|
| **Agent 异步 `Deps.Events` → `Program.Send`**（permission prompt、stream、turnDone）未列入迁移面 | P0 | FR-1.10x；DESIGN §3.10 |
| **`tuitest` harness 直连 `model.Update`**，不经 `run.go`/`safeModel`/`NewProgram` | P1 | FR-1.10x；AC-2.10 |
| **`MouseMotionMsg` 拖拽选区**（v1 `MouseActionMotion` + `MouseButtonLeft`）未单列 | P0 | 扩展 FR-1.10d/p；AC-2.3 |
| **`Init()` + `RequestWindowSize`**：v2 返回 `WindowSizeMsg`（非 Cmd）与 `tea.Batch(listenPrompt, …)` 组合模式未写清 | P0 | FR-1.10z；DESIGN §3.1 |
| **`update.go` 保留 `case tea.KeyMsg:`** 可能收到 `KeyReleaseMsg` 双触发 Enter/Esc | P0 | FR-1.10aa |
| **`passthrough.Runes` / `len(passthrough.Runes)`**（`update.go` ~L44–55）未在 FR-1.10u 点名 | P0 | 扩展 FR-1.10u |
| **`Running` 时 `updateInput` → `chatVP.Update(msg)`** 须转发 v2 键鼠 | P1 | FR-1.10ab |
| **FR-3.1 设计示例缺 `Perm: perm`**（子代理 worktree 边界） | P1 | DESIGN §5.1 |
| **`QuitAfterWait` + 双击退出** 与 v2 声明式 AltScreen 清理 | P1 | FR-3.4.3；DESIGN §3.11 |
| **测试清单遗漏** lipgloss 相关：`chattool/render_test`、`header_width_test`、`header/scroll_test`、`chat/cache_test`、`markdown/incremental_test`/`stress_test` | P2 | ACCEPTANCE §6 |
| **FR-3.1 单测落点** 应在 `spawn/execute_test.go` 而非仅 tuitest | P1 | AC-5.1 |
| **`internal/ui/clipboard/clipboard_test.go`** 迁后仍须全绿（降级路径） | P1 | ACCEPTANCE §6 |
| root **`README.md`** 未列入文档更新 | P2 | DESIGN §9 |

### 第五轮（toolVP / 双通道 / 虚拟列表 YOffset / 测试落点）

| 缺口 | 严重度 | 处置 |
|------|--------|------|
| **`toolVP` HP 与 Sync 链**：`applyViewportHP` 亦设 `toolVP.HighPerformanceRendering`；`viewportSyncCmd` batch chat+tool | P0 | 扩展 FR-1.10m；AC-2.4 |
| **Permission 双通道**：`PromptCh`+`listenPrompt()`（Cmd 闭包）与 `Events`+`p.Send` 并存；bubble 子代理权限 ask 走前者 | P0 | FR-1.10ac；DESIGN §3.10 |
| **Alt+Enter 多行输入**：`updateInput` 仍用 `key.Type==KeyEnter && !key.Alt` | P0 | FR-1.10ad |
| **drain/jump 仍依赖 HP 辅助**：`LineUp`/`LineDown` 返回值 + `viewportScrollCmdFromLines`/`viewportSyncCmdFor`；`jumpViewport` 直接 `SetYOffset` | P0 | 扩展 FR-1.10t；FR-3.7.8 |
| **`SyncChat` `atBottom`/`GotoBottom`/`SetYOffset`** 与虚拟列表 totalLines 须统一 clamp 语义 | P0 | FR-3.7.8；DESIGN §5.5 |
| **`handleNoticeScrollTick` → `withHPSync`**：header 通知滚动 sync 路径删 HP 后须改 | P1 | 扩展 FR-1.10m |
| **`WindowSizeMsg` resize**：行目录 / `headerCache` / `RenderCache` width 变化 invalidate | P1 | FR-3.7.9 |
| **`safeModel.View() string`**：inner 迁 `tea.View` 后 wrapper 须透传 `AltScreen`/`MouseMode`/`Cursor` | P0 | 扩展 FR-1.10k |
| **Bracketed paste 默认**：`PasteMsg` 依赖终端 bracketed paste；v2 经 `View.DisableBracketedPasteMode` 控制 | P1 | 扩展 FR-1.10o |
| **`lipgloss.SetColorProfile`**（`markdown/render_test.go`）v2 测试 API 对齐 | P1 | FR-1.10ae |
| **影响面遗漏 9 文件**：`deps/`、`style/`、`layout/`、`chattool/styles`、`*update.go`（turn/subagent/session/overlay） | P1 | DESIGN §4 |
| **`viewport_hp_test.go` 应改写非删除**：断言 HP sync → `scheduleSyncChatView` 或 sync 语义 | P1 | ACCEPTANCE §6 |
| **测试清单遗漏**：`history_test`、`turn/update_test`/`cancel_test`/`blocks_test`、`turn_metrics_test`、`subagent/registry_test` | P2 | ACCEPTANCE §6 |

**不在 v0.1.3 范围**（保持 v0.1.2 非目标）：Kitty 键盘增强全量绑定、MCP spill GC、`@` compact 专用脱敏、`View.OnMouse` 拦截、transcript/classic 模式（FR-3.5）。

## 已知限制

| 限制 | 说明 |
|------|------|
| **Breaking TUI 内部实现** | 对外用户行为力求等价；HP/`SyncScrollArea`/`withHPSync` 删除；聊天区改 **虚拟列表 + Cursed Renderer**（FR-3.7） |
| **v1 不再支持** | v0.1.3 起全仓库 `.go` 与 `go.mod`/`go.sum` 不应残留 `github.com/charmbracelet/bubbletea` v1.x |
| **流式选区错位** | FR-7.9 保留 |
| **Windows** | release 矩阵仍与 v0.1.2 一致；v2 改善 Windows 输入，可选手动矩阵 |
| **P2 延期项** | transcript/classic 未实现时行为与 v0.1.2 相同 |
| **CI / release 无 test-tui** | GitHub Actions 与 release workflow 当前不跑 `make test-tui`；发布前须本地执行（NFR-8） |
| **FR-7.14 用户语义** | v2 将 AltScreen 迁入 `View.AltScreen`，**用户仍**全屏 TUI；非 classic/transcript 模式 |
| **verify-charm-v2** | 已纳入 Makefile、`verify-release` 与 CI `test` job |

## 依赖与前置

- 基线：v0.1.2 已发布或合入 main
- Go 1.26+；通读 [Bubble Tea v2 Upgrade Guide](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md)
- 实现顺序：**go.mod v2** → **View/消息层骨架** → 编译修复 → 滚动/选区 → 单测 → 延期项 → 手动矩阵（DESIGN §6）
- 上游参考：[v2.0.0 Release Notes](https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0)

## 关联文档

- 上一版本：[../v0.1.2/README.md](../v0.1.2/README.md)
- TUI 集成测试：[../v0.1.0/TUI_INTEGRATION_TEST.md](../v0.1.0/TUI_INTEGRATION_TEST.md)
- 安全基线：[../v0.1.0/SECURITY.md](../v0.1.0/SECURITY.md)
