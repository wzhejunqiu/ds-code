# v0.1.3 需求文档

> 版本：v0.1.3  
> 状态：规划中  
> 更新日期：2026-06-20  
> 审核：2026-06-20（五轮）

## 1. 目标

1. **Bubble Tea v2 一次性迁移**：将 TUI 栈迁至 `charm.land/bubbletea/v2` 及配套的 `bubbles`/`lipgloss`/`glamour` v2（见 FR-1）；**不**经过 v1.3.x 中间版本。
2. **迁移后行为等价**：v0.1.2 已交付的交互 TUI 能力须通过自动化与手动回归；允许内部实现替换（如 Cursed Renderer 取代 HP），**不得**引入可见功能回退（见 FR-2）。
3. **历史延期项补全 + 长 transcript 性能**：v0.1.0–v0.1.2 延期条目（见 FR-3）；**实现** v0.1.2 曾拒绝的聊天区虚拟列表（FR-3.7 / 原 FR-9.12）。
4. **技术债清理**：移除导出 `permission.IsSensitiveAbs`（见 FR-4）。

**非目标**：保留 v1 兼容层或双栈编译；权限/MCP/Agent 核心逻辑变更；新 LLM Provider；Windows 正式 release 矩阵扩展。

## 2. 用户故事

### US-1：一次升级，长期对齐 Charm 生态

**作为** 维护者，  
**我希望** ds-code 直接使用 Bubble Tea v2 与 `charm.land` 模块路径，  
**以便** 避免 v1→v2 二次大改，并获得 Cursed Renderer、Unicode mode 2027、内置剪贴板等能力。

**验收**：`go.mod` 仅 require v2 路径（FR-1）；无 `github.com/charmbracelet/bubbletea` v1 import。

### US-2：升级后 TUI 手感不退化

**作为** 开发者，  
**我希望** 迁 v2 后滚轮平滑、选区复制、Markdown、输入框与 v0.1.2 一致或更好，  
**以便** 日常 Agent 交互不受版本升级影响。

**验收**：ACCEPTANCE §3–§5；长 transcript 滚轮 **不劣于** v0.1.2（v2 渲染器允许 **优于** 基线）。

### US-3：SSH 下复制更可靠（v2 能力）

**作为** 通过 SSH 使用 ds-code 的用户，  
**我希望** 选区复制优先走 Bubble Tea 原生 OSC52（`tea.SetClipboard`），  
**以便** 减少「无 pbcopy/xclip」时的复制失败。

**验收**：FR-1.9；SSH 终端 smoke（ACCEPTANCE §4）。

### US-7：极长 transcript 仍流畅

**作为** 长时间使用 ds-code 的开发者，  
**我希望** 聊天区在数百轮对话后滚轮与流式更新仍流畅，  
**以便** 不必因全量重绘卡顿而中断 Agent 交互。

**验收**：FR-3.7；AC-5.5；NFR-2 扩展。

### US-4–US-6

与 v0.1.2 规划相同：浮层选区（US-3/FR-3.2）、子代理 `@`（US-4/FR-3.1）、选区增强（US-5/FR-3.3）、`IsSensitiveAbs` 删除（US-6/FR-4）。

## 3. 功能需求

### FR-1 Bubble Tea v2 生态升级

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-1.1 | `charm.land/bubbletea/v2` → **v2.0.7** 或 tidy 解析的最新 v2 补丁 | P0 |
| FR-1.2 | `charm.land/bubbles/v2` → **v2.1.0+** | P0 |
| FR-1.3 | `charm.land/lipgloss/v2` → **v2.0.4+** | P0 |
| FR-1.4 | `charm.land/glamour/v2` → **v2.0.1+** | P0 |
| FR-1.5 | `github.com/charmbracelet/x/ansi` → **v0.11.7+** | P1 |
| FR-1.6 | `go mod tidy`；移除全部 v1 线 `github.com/charmbracelet/bubbletea`、`bubbles` v0.x/v1.0 直接 require | P0 |
| FR-1.7 | 全仓库 TUI import 统一为 `charm.land/*/v2`；**禁止**残留 v1 bubbletea import | P0 |
| FR-1.8 | 按 [Upgrade Guide](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md) 完成 API 迁移（见 FR-1.10） | P0 |
| FR-1.9 | 评估 `tea.SetClipboard` / `tea.ReadClipboard` 作为 `internal/ui/clipboard` 首选路径；外部命令作降级 | P1 |
| FR-1.10 | **迁移清单**（P0 子项）： | P0 |
| FR-1.10a | `View() string` → `View() tea.View`；内容经 `v.SetContent(...)` 或 `tea.NewView` | P0 |
| FR-1.10b | `tea.WithAltScreen()` / `WithMouseCellMotion()` → `View.AltScreen`、`View.MouseMode` | P0 |
| FR-1.10c | `tea.KeyMsg` → `tea.KeyPressMsg`（`msg.String()` 匹配；空格 `"space"`） | P0 |
| FR-1.10d | `tea.MouseMsg` → `MouseClickMsg` / `MouseReleaseMsg` / `MouseWheelMsg` / `MouseMotionMsg` | P0 |
| FR-1.10e | 移除 v1 专属 Cmd：`EnterAltScreen`、`SyncScrollArea`、`EnableMouseCellMotion` 等 | P0 |
| FR-1.10f | `tea.WindowSize()` → `tea.RequestWindowSize`；Init 中请求尺寸 | P0 |
| FR-1.10g | `bubbles/viewport`、`textinput` 等改 import `charm.land/bubbles/v2/...` | P0 |
| FR-1.10h | **键位全量迁移**：`tea.KeyType`/`key.Type`/`key.Matches` → `KeyPressMsg.String()` 或 bubbles v2 `key.Matches`；所有 `func(tea.KeyMsg)` 回调签名与单测构造器同步改（overlay、picker、subagent、input、wheel_scroll） | P0 |
| FR-1.10i | `glamour/ansi`、`glamour/styles` 等子包改 `charm.land/glamour/v2/...`；`markdown/styles.go` 样式 API 对齐 v2；`mdProfile` 缓存键与 `lipgloss.ColorProfile()` v2 返回值比较方式对齐（可能非 `termenv.Profile` 直接 `==`） | P0 |
| FR-1.10j | `input/mouse_escape.go`：SGR 泄漏恢复适配 v2（`KeyPressMsg.Text` 字符流 → `MouseWheelMsg`/`MouseClickMsg`；`passthrough` 改 `KeyPressMsg`；**不再**构造 v1 struct `MouseMsg`/`KeyRunes`）；迁后 iTerm2 快速滚轮无输入框乱码 | P0 |
| FR-1.10k | `safe_model.go`：`View() tea.View` 包装层；从 `inner.View()` **透传** `AltScreen`/`MouseMode`/`Cursor`/`Content`；`fallbackView()` 亦返回 `tea.View` 并设 `AltScreen`/`MouseMode`；panic fallback 行为不退化 | P0 |
| FR-1.10l | `scroll/wheel.go`：`ComputeWheelStep`/`wheelDirection` 改接 `MouseWheelMsg`（不再用 `MouseButtonWheelUp/Down`） | P0 |
| FR-1.10m | 删除 `viewport_hp.go` 及 `withHPSync`/`applyViewportHP`/`syncChatViewportHP`/`viewportSyncCmd*` 调用链（`update.go` **16** 处 + `ticks.go` 1 处）；**含** `toolVP.HighPerformanceRendering` 与 `viewportSyncCmd` batch tool Sync；`handleNoticeScrollTick` 不再 `withHPSync`；`selDragging` 时改由可见窗口 refresh 保证选区高亮；`selection_update` 拖拽开始**不得**再调 `applyViewportHP`（DESIGN §3.4） | P0 |
| FR-1.10n | `Model.Init()` 与 `safeModel.Init()` 返回 `tea.RequestWindowSize`（v2 无 `tea.WindowSize()` Cmd；语义见 DESIGN §3.1） | P0 |
| FR-1.10o | **`tea.PasteMsg`**：粘贴不再经 `KeyMsg.Paste`/`Runes`；`update.go`/`updateInput` 显式处理 `PasteMsg`（及可选 `PasteStartMsg`/`PasteEndMsg`），再转发 bubbles `textinput`；**默认**保持 bracketed paste 启用（`View.DisableBracketedPasteMode=false`），大段粘贴回归 | P0 |
| FR-1.10p | **鼠标常量与坐标**：`MouseButtonLeft`→`MouseLeft`；`MouseEvent(msg).IsWheel()`→类型断言或 `MouseWheelMsg`；坐标 `msg.X/Y`→`msg.Mouse().X/Y`（`mapMousePoint`、`handleMouseWheel` 分区滚轮） | P0 |
| FR-1.10q | **`View()` 去副作用**：`model.go` 中 `applyViewportHP()`、惰性 `plainLines` 重建移出 `View()`，改在 `Update`/sync 路径预计算；`View()` 只组装 `tea.View` | P0 |
| FR-1.10r | **build tag 分裂文件**同步迁 v2：`input_release.go`、`input_debug.go`、`submit_hook_release.go`、`submit_hook_tuitest.go`、`picker_release.go`、`picker_tuitest.go` | P1 |
| FR-1.10s | **`viewportHPEnabled()` 逻辑拆分**：除 HP 外还控制浮层/权限 prompt 禁滚轮与禁选区；删 HP 后须保留为 `chatInteractionEnabled()`（或等价），**不得**与 HP 字段同删 | P0 |
| FR-1.10t | **`wheel_scroll.go` 去 HP 耦合**：`handleWheelScrollTick`/`jumpViewport` 不再调 `viewportScrollCmdFromLines`/`viewportSyncCmdFor`；drain 仅用 `LineUp`/`LineDown` + `SetYOffset`（或 model 层 YOffset）+ `scheduleSyncChatView`；`jumpViewport` 的 pending 合并逻辑保留 | P0 |
| FR-1.10u | **`mouse_escape` 全链路**：`RecoverLeakedMouseKeys`/`AccumulateLeakedMouseKeys`/`extractLeakedSGREvents` 输出 v2 `MouseWheelMsg` 等（非 v1 struct）；passthrough 用 `KeyPressMsg.Text`（**非** `Runes`）；`update.go` 中 `len(passthrough.Runes)`/`passthrough.Runes` 须改为 `Text`（~L44–55）；循环分派改接 v2 类型 | P0 |
| FR-1.10v | **`asyncCopy` 与 `tea.SetClipboard`**：评估将 `selection_update.asyncCopy` 从 goroutine+`clipboard.Write` 改为 `Update` 返回 `tea.SetClipboard`（失败仍走 `internal/ui/clipboard` + `copyResultMsg` toast） | P1 |
| FR-1.10w | **bubbles textinput v2 光标**：评估 v2 是否要求 `tea.View.Cursor` 而非 lipgloss 内嵌光标；`view/render.go` 输入区与 `Model.View()` 组装须对齐 | P1 |
| FR-1.10x | **异步事件通道**：`run.go` 中 `Deps.Events` + goroutine `p.Send(msg)`（agent stream、permission prompt、`turnDone` 等）迁后仍可用；`internal/tuitest` harness **不经** `Program`，直连 `model.Update`——须同步 v2 消息构造（DESIGN §3.10） | P0 |
| FR-1.10y | **`MouseMotionMsg` 拖拽**：`selection_update` 中 v1 `MouseActionMotion` + `MouseButtonLeft` 改为 v2 `MouseMotionMsg`（`msg.Button == tea.MouseLeft`）；坐标 `msg.Mouse().X/Y` | P0 |
| FR-1.10z | **`Init()` + `RequestWindowSize`**：v2 `RequestWindowSize` 返回 `WindowSizeMsg`（**非** `tea.Cmd`）；与 `tea.Batch(listenPrompt, LoadInitialHistory, …)` 组合策略须明确（DESIGN §3.1） | P0 |
| FR-1.10aa | **`KeyReleaseMsg` 排除**：`update.go` 优先 `case tea.KeyPressMsg:`；若保留 `case tea.KeyMsg:` 须在分支内 type-switch **忽略** `KeyReleaseMsg`，避免 Enter/Esc/Ctrl+C 双触发 | P0 |
| FR-1.10ab | **`Running` 模式 viewport 转发**：`updateInput` 在 `m.Running` 时 `chatVP.Update(msg)` 须接收 v2 `KeyPressMsg`/`MouseWheelMsg` 等（PgUp/PgDn 滚动历史） | P1 |
| FR-1.10ac | **Permission 双通道**：`run.go` 中 `Events` goroutine `p.Send` **与** `listenPrompt()`（`PromptCh` → Cmd 闭包 → `PromptRequestMsg`）并存；bubble 子代理权限 ask 走 `PromptCh`；迁 v2 后两路径均须回归 | P0 |
| FR-1.10ad | **Alt+Enter 多行**：`updateInput` Enter 提交判定 `key.Type==KeyEnter && !key.Alt` → `KeyPressMsg.String()=="enter" && !msg.Mod.Contains(tea.ModAlt)`（或等价 `msg.Code`/`msg.Mod` 组合） | P0 |
| FR-1.10ae | **测试色板 API**：`markdown/render_test.go` 等 `lipgloss.SetColorProfile(termenv.TrueColor)` 对齐 lipgloss v2 / `tea.WithColorProfile`（tuitest 确定性渲染） | P1 |
| FR-1.11 | v0.1.2 `viewport_hp.go`（HP + SyncScrollArea）：**删除**；滚动语义由 `scroll/` pending/drain + v2 渲染器承担（DESIGN §3） | P0 |
| FR-1.12 | 升级后修复全部编译与 `-race` 单测；**不**借迁移无关重构 Agent 层 | P0 |
| FR-1.13 | `go mod tidy` 后 `go.sum` **无** v1 charm 模块行；`verify-charm-v2` grep **全部** `github.com/charmbracelet/{bubbletea,bubbles,lipgloss,glamour}`；评估 `muesli/termenv` 与 lipgloss v2 `ColorProfile()` 兼容性 | P0 |

### FR-2 迁移后回归

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-2.1 | `make test` 全绿 | P0 |
| FR-2.2 | `make test-tui` 全绿 | P0 |
| FR-2.3 | `make lint` / `make vet` 无新增失败 | P0 |
| FR-2.4 | `make verify-release` 通过 | P0 |
| FR-2.5 | v0.1.2 AC §7–§8 TUI 选区与平滑滚动 **语义**等价 | P0 |
| FR-2.6 | v0.1.2 AC §11 手动项（TUI 必跑；非 TUI 抽样） | P1 |
| FR-2.7 | 更新 TUI 单测：v2 消息类型、wheel drain、selection、mouse_escape、glamour smoke | P1 |
| FR-2.8 | 终端矩阵：macOS 原生 + Cursor/VS Code 集成 + 可选 Ghostty/iTerm2 + SSH smoke | P1 |
| FR-2.9 | **保留** v0.1.2 已实现的 `DS_CODE_SCROLL_SPEED` 与 `scroll.DetectProfile()`（integrated vs native drain）；v2 迁移不得回退 | P0 |
| FR-2.10 | iTerm2 SGR 鼠标泄漏回归：`mouse_escape` 路径 + 输入框无 `[<64;…M` 乱码（[v0.1.2 AC 手动 #41](../v0.1.2/ACCEPTANCE.md)） | P0 |
| FR-2.11 | `scroll.DetectProfile()`（`TERM_PROGRAM`/`VSCODE_INJECTED`）与 `DS_CODE_SCROLL_SPEED` 单测或 tuitest smoke；集成终端 adaptive、原生 proportional 不退化（[v0.1.2 FR-9.8](../v0.1.2/REQUIREMENTS.md)） | P1 |

### FR-3 历史延期项补全

| ID | 描述 | 优先级 | 落点 |
|----|------|--------|------|
| FR-3.1 | 子代理 prompt `@` 展开：`spawn/execute.go` 子 `ctxSvc` 注入 `AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}`（与主会话 `app/runner.go` 一致；`perm` 为子代理 `Engine`） | P1 | `execute.go` ~L99 `ctxSvc` 组装 |
| FR-3.2 | 浮层选区（`/help`、权限 prompt）：打开浮层时禁用聊天选区（v0.1.2 首期）；完整版可在浮层内选区复制 | P1 | `selection_update.go`、`overlay/*` |
| FR-3.3 | 选区增强：双击选词、三击选行；`Shift+↑/↓/Home/End` 键盘扩展 | P1–P2 | `MouseClickMsg` 时间戳；`KeyPressMsg` |
| FR-3.7 | **TUI 聊天区虚拟列表**（承接 v0.1.2 FR-9.12「不实现」→ v0.1.3 **实现**）：见 FR-3.7.1–3.7.7 | P1 | `chat/`、`model/view/render.go`、`sync.go` |
| FR-3.5 | Transcript scrollback / classic 无 AltScreen 渲染器 | P2 | 可选；v2 `View.AltScreen` 更易预埋 |
| FR-3.6 | `LoadSkill` skill 名 `..` 拦截 — **保持现状**，不纳入本版 | — | Out of scope |

#### FR-3.4 退出与中断（v2）

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-3.4.1 | 空闲态全局退出：`tea.Quit` / v2 等价路径；声明式 `View.AltScreen=false` 由框架清理 | P0 |
| FR-3.4.2 | Agent 回合 Esc → `RequestCancelTurn`；**不**误触全局 Quit | P0 |
| FR-3.4.3 | 双击 Ctrl+C 退出后终端无 alt screen 残留 | P1 |

#### FR-3.7 聊天区虚拟列表（原 v0.1.2 FR-9.12）

> **背景**：v0.1.2 `chat.RenderCache` 仅缓存 **block** 级渲染行，但 `SyncChat` 仍将 header+全部 block **拼接为整段字符串** 后 `chatVP.SetContent`；删 HP 迁 v2 后，极长 transcript 的全量拼接 + lipgloss 重绘成本随总行数线性增长。虚拟列表只向 viewport 提交 **可见行窗口**（+ 可选 overscan），plain 行索引与选区坐标系保持全局一致。

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-3.7.1 | **保留** block 级 `RenderCache` / `markdown.SegmentCache`；新增 **行级目录**（header 行 + 各 block 展开后的 plain/styled 行数、块→行偏移） | P1 |
| FR-3.7.2 | `SyncChat`（及 drain 结束后的 flush）向 `chatVP.SetContent` 仅写入 `[YOffset, YOffset+Height)` 对应 **styled** 切片（DESIGN §5.5） | P1 |
| FR-3.7.3 | **总行数**、`chatVP.YOffset`、滚轮 pending/drain、PgUp/PgDn、工具面板路由与 v0.1.2 **语义一致**；虚拟列表为内部优化，用户不可见行为变化 | P0 |
| FR-3.7.4 | **选区 / 复制**：维护全量 **plain** 行切片（`plainLines` 或等价结构），供 `selection.Extract` / `mapMousePoint`；**不要求**维护全量 styled 字符串 | P1 |
| FR-3.7.5 | **流式更新**：仅 invalidate 尾部 block（及 header 若变）的行目录；drain 活跃期间（FR-9.7 语义）不重建可见窗口外的 styled 内容 | P1 |
| FR-3.7.6 | **选区高亮**：`visibleHighlightedLines` 仍在可见窗口内 overlay；`selDragging` 时可见窗口全量 refresh（衔接 FR-1.10q，**不**回退全 transcript styled 拼接） | P1 |
| FR-3.7.7 | 单测 / benchmark：合成 ≥500 行 transcript，`syncChatView` 或可见窗口 rebuild 耗时 **不随总行数线性增长**（允许随 viewport 高度线性） | P1 |
| FR-3.7.8 | **`jumpViewport` / drain YOffset**：`wheel_scroll.jumpViewport` 与 `SyncChat` 的 `atBottom`/`GotoBottom`/`SetYOffset` 在虚拟列表下统一由 model/`LineCatalog` 维护 `totalLines` 并 clamp `[0, totalLines-height]`；**禁止** YOffset 漂移导致空白或截断 | P0 |
| FR-3.7.9 | **`WindowSizeMsg` resize**：width/height 变化时 invalidate `LineCatalog`、`headerCache`、`RenderCache`（width 键）；`overlay.OnWindowSize` 后可见窗口 rebuild；选区 `mapMousePoint` 坐标仍正确 | P1 |

### FR-4 移除 Deprecated `IsSensitiveAbs`

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-4.1 | `IsSensitiveAbs` 改为包内 `isSensitiveAbs`（或删除导出）；`path_api.go` 等包内调用同步改名；对外仅 `SkipSensitiveAbs` / `ResolveAccessPath` | P0 |
| FR-4.2 | `permission/sensitive_test.go` 改测 `SkipSensitiveAbs` 或包内测试；**无**包外 `permission.IsSensitiveAbs` 引用 | P0 |
| FR-4.3 | `globmatch`、builtin tools 等保持 v0.1.2 已收敛状态（无回退） | P0 |

## 4. 非功能需求

| ID | 描述 |
|----|------|
| NFR-1 | v2 迁移（FR-1）与 FR-2 为 **P0 发布门槛**；FR-3 P1（含 **FR-3.7**）同版本交付；未合入须在 CHANGELOG 说明 |
| NFR-2 | 长 transcript：滚轮 p95 **不劣于** v0.1.2；虚拟列表（FR-3.7）下 `syncChatView` **不**随总行数线性恶化；v2 Cursed Renderer 允许更快 |
| NFR-3 | 二进制体积 ±15% 内（v2 渲染栈略增可接受） |
| NFR-4 | CI / Makefile 增加 v1 import 守卫（DESIGN §10） |
| NFR-5 | 新增/改动的 TUI 行为须有单测或 tuitest 覆盖 |
| NFR-6 | 全仓库 **66** 个含 `charmbracelet` import 的 `.go` 文件须完成 v2 路径替换（含 `internal/logging/warn.go`） |
| NFR-7 | `rg 'github.com/charmbracelet/(bubbletea|bubbles|lipgloss|glamour)' --glob '*.go' --glob 'go.mod' .` 无匹配（注释除外） |
| NFR-8 | 发布前本地执行 `make test-tui`（CI / `release.yml` 当前均未覆盖） |
| NFR-9 | `verify-release` 或 release 流程纳入 `verify-charm-v2`（与 NFR-4 一致；当前 Makefile **尚无**该 target，实现时新增） |

## 5. 范围边界

**In scope**

- 全部 `internal/ui/**`（含 `theme/colors.go`、`selection/selection.go`、`style/`、`layout/`、`deps/deps.go`）与 TUI 相关 `internal/logging/warn.go`（lipgloss）
- `internal/ui/clipboard/**`（FR-1.9 `tea.SetClipboard` 评估；`clipboard_test.go` 降级路径）
- `go.mod` / `go.sum`
- `internal/agent/spawn/execute.go`（FR-3.1）
- `internal/agent/spawn/execute_test.go`（FR-3.1 `@` 单测）
- `internal/permission/sensitive.go`、`path_api.go`（FR-4）
- `internal/tuitest/**`（harness 直连 `model.Update`，见 FR-1.10x）
- `docs/v0.1.3/**`、`CHANGELOG.md`、`README.md`（若提及 Charm 版本）、`internal/ui/tui/README.md`、`CLAUDE.md`
- `.github/workflows`（`ci.yml` / `release.yml`）或 `Makefile`：v1 import 守卫（NFR-4、NFR-9）
- `docs/v0.1.0/CONFIG.md`、`docs/v0.1.0/SECURITY-SYNC.md`（若剪贴板/OSC52 行为变更）

**Out of scope**

- v1.3.x 中间版本或 v1/v2 双栈
- Agent Runner / MCP / S2/S3 策略（除 FR-4）
- Windows release artifact 新增
- spill GC、web_search 实现等

## 6. 实现优先级建议

| 阶段 | 内容 |
|------|------|
| **A** | `go.mod` → charm.land v2；import 批量替换（66 文件） |
| **B** | `View() tea.View` 骨架 + `run.go` + `safe_model` + `Init`/`RequestWindowSize`；**View 去副作用**（FR-1.10q） |
| **C** | `update.go` 键鼠迁移；`PasteMsg`；`mouse_escape`；`KeyPressMsg`/`MouseMotionMsg` 全仓库；排除 `KeyReleaseMsg`（FR-1.10aa） |
| **D** | 滚动：删 HP/`withHPSync`/`viewportScrollCmdFromLines`；`scroll/wheel.go` + `wheel_scroll.go` 接 `MouseWheelMsg`（FR-1.10t） |
| **E** | 选区 + clipboard（含 `tea.SetClipboard` 评估） |
| **F** | `make test` + `make test-tui` 全绿 |
| **G** | FR-4 + FR-3 延期项 + **FR-3.7 虚拟列表**（建议在删 HP / 回归通过后实施，DESIGN §5.5） |
| **H** | 手动矩阵 + CHANGELOG + v1 守卫脚本 |
