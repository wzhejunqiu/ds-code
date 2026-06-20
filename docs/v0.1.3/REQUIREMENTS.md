# v0.1.3 需求文档

> 版本：v0.1.3  
> 状态：规划中  
> 更新日期：2026-06-20  
> 审核：2026-06-20（三轮）

## 1. 目标

1. **Bubble Tea v2 一次性迁移**：将 TUI 栈迁至 `charm.land/bubbletea/v2` 及配套的 `bubbles`/`lipgloss`/`glamour` v2（见 FR-1）；**不**经过 v1.3.x 中间版本。
2. **迁移后行为等价**：v0.1.2 已交付的交互 TUI 能力须通过自动化与手动回归；允许内部实现替换（如 Cursed Renderer 取代 HP），**不得**引入可见功能回退（见 FR-2）。
3. **历史延期项补全**：v0.1.0–v0.1.2 标记延期的条目（见 FR-3）。
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
| FR-1.10i | `glamour/ansi`、`glamour/styles` 等子包改 `charm.land/glamour/v2/...`；`markdown/styles.go` 样式 API 对齐 v2 | P0 |
| FR-1.10j | `input/mouse_escape.go`：SGR 泄漏恢复适配 v2（`KeyPressMsg.Text` 字符流 → `MouseWheelMsg`/`MouseClickMsg`；`passthrough` 改 `KeyPressMsg`；**不再**构造 v1 struct `MouseMsg`/`KeyRunes`）；迁后 iTerm2 快速滚轮无输入框乱码 | P0 |
| FR-1.10k | `safe_model.go`：`View() tea.View` 包装层；`fallbackView()` 亦返回 `tea.View` 并设 `AltScreen`/`MouseMode`；panic fallback 行为不退化 | P0 |
| FR-1.10l | `scroll/wheel.go`：`ComputeWheelStep`/`wheelDirection` 改接 `MouseWheelMsg`（不再用 `MouseButtonWheelUp/Down`） | P0 |
| FR-1.10m | 删除 `viewport_hp.go` 及 `withHPSync`/`applyViewportHP`/`syncChatViewportHP` 调用链（`update.go` **16** 处 + `ticks.go` 1 处）；`selDragging` 时改由全量 `SetContent` 保证选区高亮；`selection_update` 拖拽开始**不得**再调 `applyViewportHP`（DESIGN §3.4） | P0 |
| FR-1.10n | `Model.Init()` 与 `safeModel.Init()` 返回 `tea.RequestWindowSize`（v2 无 `tea.WindowSize()` Cmd；语义见 DESIGN §3.1） | P0 |
| FR-1.10o | **`tea.PasteMsg`**：粘贴不再经 `KeyMsg.Paste`/`Runes`；`update.go`/`updateInput` 显式处理 `PasteMsg`（及可选 `PasteStartMsg`/`PasteEndMsg`），再转发 bubbles `textinput` | P0 |
| FR-1.10p | **鼠标常量与坐标**：`MouseButtonLeft`→`MouseLeft`；`MouseEvent(msg).IsWheel()`→类型断言或 `MouseWheelMsg`；坐标 `msg.X/Y`→`msg.Mouse().X/Y`（`mapMousePoint`、`handleMouseWheel` 分区滚轮） | P0 |
| FR-1.10q | **`View()` 去副作用**：`model.go` 中 `applyViewportHP()`、惰性 `plainLines` 重建移出 `View()`，改在 `Update`/sync 路径预计算；`View()` 只组装 `tea.View` | P0 |
| FR-1.10r | **build tag 分裂文件**同步迁 v2：`input_release.go`、`input_debug.go`、`submit_hook_release.go`、`submit_hook_tuitest.go`、`picker_release.go`、`picker_tuitest.go` | P1 |
| FR-1.10s | **`viewportHPEnabled()` 逻辑拆分**：除 HP 外还控制浮层/权限 prompt 禁滚轮与禁选区；删 HP 后须保留为 `chatInteractionEnabled()`（或等价），**不得**与 HP 字段同删 | P0 |
| FR-1.10t | **`wheel_scroll.go` 去 HP 耦合**：`handleWheelScrollTick`/`jumpViewport` 不再调 `viewportScrollCmdFromLines`/`viewportSyncCmdFor`（现位于 `viewport_hp.go`）；drain 仅 `LineUp`/`LineDown` + `scheduleSyncChatView` | P0 |
| FR-1.10u | **`mouse_escape` 全链路**：`RecoverLeakedMouseKeys`/`AccumulateLeakedMouseKeys`/`extractLeakedSGREvents` 输出 v2 `MouseWheelMsg` 等（非 v1 struct）；passthrough 用 `KeyPressMsg.Text`（**非** `Runes`）；`update.go` 循环分派改接 v2 类型 | P0 |
| FR-1.10v | **`asyncCopy` 与 `tea.SetClipboard`**：评估将 `selection_update.asyncCopy` 从 goroutine+`clipboard.Write` 改为 `Update` 返回 `tea.SetClipboard`（失败仍走 `internal/ui/clipboard` + `copyResultMsg` toast） | P1 |
| FR-1.10w | **bubbles textinput v2 光标**：评估 v2 是否要求 `tea.View.Cursor` 而非 lipgloss 内嵌光标；`view/render.go` 输入区与 `Model.View()` 组装须对齐 | P1 |
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
| FR-3.1 | 子代理 prompt `@` 展开：`spawn/execute.go` 子 `ctxSvc` 注入 `AtExpander`（与主会话 `app/runner.go` 一致） | P1 | `execute.go` ~L100 `ctxSvc` 组装 |
| FR-3.2 | 浮层选区（`/help`、权限 prompt）：打开浮层时禁用聊天选区（v0.1.2 首期）；完整版可在浮层内选区复制 | P1 | `selection_update.go`、`overlay/*` |
| FR-3.3 | 选区增强：双击选词、三击选行；`Shift+↑/↓/Home/End` 键盘扩展 | P1–P2 | `MouseClickMsg` 时间戳；`KeyPressMsg` |
| FR-3.5 | Transcript scrollback / classic 无 AltScreen 渲染器 | P2 | 可选；v2 `View.AltScreen` 更易预埋 |
| FR-3.6 | `LoadSkill` skill 名 `..` 拦截 — **保持现状**，不纳入本版 | — | Out of scope |

#### FR-3.4 退出与中断（v2）

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-3.4.1 | 空闲态全局退出：`tea.Quit` / v2 等价路径；声明式 `View.AltScreen=false` 由框架清理 | P0 |
| FR-3.4.2 | Agent 回合 Esc → `RequestCancelTurn`；**不**误触全局 Quit | P0 |
| FR-3.4.3 | 双击 Ctrl+C 退出后终端无 alt screen 残留 | P1 |

### FR-4 移除 Deprecated `IsSensitiveAbs`

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-4.1 | `IsSensitiveAbs` 改为包内 `isSensitiveAbs`（或删除导出）；`path_api.go` 等包内调用同步改名；对外仅 `SkipSensitiveAbs` / `ResolveAccessPath` | P0 |
| FR-4.2 | `permission/sensitive_test.go` 改测 `SkipSensitiveAbs` 或包内测试；**无**包外 `permission.IsSensitiveAbs` 引用 | P0 |
| FR-4.3 | `globmatch`、builtin tools 等保持 v0.1.2 已收敛状态（无回退） | P0 |

## 4. 非功能需求

| ID | 描述 |
|----|------|
| NFR-1 | v2 迁移（FR-1）与 FR-2 为 **P0 发布门槛**；FR-3 P1 可文档化延期但须在 CHANGELOG 说明 |
| NFR-2 | 长 transcript 滚轮：p95 **不劣于** v0.1.2；v2 Cursed Renderer 允许更快 |
| NFR-3 | 二进制体积 ±15% 内（v2 渲染栈略增可接受） |
| NFR-4 | CI / Makefile 增加 v1 import 守卫（DESIGN §10） |
| NFR-5 | 新增/改动的 TUI 行为须有单测或 tuitest 覆盖 |
| NFR-6 | 全仓库 **66** 个含 `charmbracelet` import 的 `.go` 文件须完成 v2 路径替换（含 `internal/logging/warn.go`） |
| NFR-7 | `rg 'github.com/charmbracelet/(bubbletea|bubbles|lipgloss|glamour)' --glob '*.go' --glob 'go.mod' .` 无匹配（注释除外） |
| NFR-8 | 发布前本地执行 `make test-tui`（CI / `release.yml` 当前均未覆盖） |
| NFR-9 | `verify-release` 或 release 流程纳入 `verify-charm-v2`（与 NFR-4 一致；当前 Makefile **尚无**该 target，实现时新增） |

## 5. 范围边界

**In scope**

- 全部 `internal/ui/**`（含 `theme/colors.go`、`selection/selection.go`）与 TUI 相关 `internal/logging/warn.go`（lipgloss）
- `internal/ui/clipboard/**`（FR-1.9 `tea.SetClipboard` 评估）
- `go.mod` / `go.sum`
- `internal/agent/spawn/execute.go`（FR-3.1）
- `internal/permission/sensitive.go`、`path_api.go`（FR-4）
- `internal/tuitest/**`
- `docs/v0.1.3/**`、`CHANGELOG.md`、`internal/ui/tui/README.md`、`CLAUDE.md`
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
| **C** | `update.go` 键鼠迁移；`PasteMsg`；`mouse_escape`；`KeyPressMsg` 全仓库 |
| **D** | 滚动：删 HP/`withHPSync`/`viewportScrollCmdFromLines`；`scroll/wheel.go` + `wheel_scroll.go` 接 `MouseWheelMsg`（FR-1.10t） |
| **E** | 选区 + clipboard（含 `tea.SetClipboard` 评估） |
| **F** | `make test` + `make test-tui` 全绿 |
| **G** | FR-4 + FR-3 延期项 |
| **H** | 手动矩阵 + CHANGELOG + v1 守卫脚本 |
