# v0.2.0 需求文档（设计总纲）

> 位置：`docs/v0.2.0/spec/`
> 版本：v0.2.0-spec
> 状态：设计中（Design）
> 更新日期：2026-07-04
> 实现版本：从 [v0.2.0 根](../)（M0）起，见 [ROADMAP.md](ROADMAP.md)
> 基线：v0.1.5
> 目标形态：macOS 桌面应用（Go + Wails v3 + TypeScript）

## 1. 目标

1. **交付一套完整的 macOS 桌面应用产品设计（P0）**：覆盖 workspace 管理、会话、聊天/流式、工具卡片、权限、设置、系统集成，面向 CLI/TUI 功能对齐。
2. **遵循 macOS 桌面设计哲学（P0）**：原生窗口/菜单栏/快捷键、克制视觉与动效、系统集成（通知、Dock、拖拽、托盘），符合 macOS 用户直觉。
3. **UI 全新设计、动线不照搬 TUI（P0）**：以可见控件与指针交互为主，键盘为辅；**取消 turn 使用可见「停止」按钮而非 ESC**；不复刻 TUI 的模式化按键序列。
4. **modal-free 优先（P0）**：权限、设置、命令、onboarding 尽量内联；避免打断式弹窗，仅系统级不可避免场景使用系统原生面板。
5. **Workspace 与 Agent 对话窗口（P0）**：workspace 关联 project（`ProjectRoot`）；其下可创建**多个 Agent 对话窗口**（对应 TUI session），UI 上作为 workspace 子项；单窗口 + 左侧栏管理多个 workspace。
6. **数据与 CLI/TUI 隔离（P0）**：`ProjectID` 算法**不变**；桌面数据落在 `~/.ds-code/desktop/projects/<project-id>/`，与 CLI/TUI 的 `~/.ds-code/projects/<project-id>/` 靠目录前缀区隔，**不共享** sessions/checkpoint/audit 等运行时数据。
7. **最大化复用 Go 核心（P0）**：仅新增桌面适配层（`cmd/ds-code-desktop`、`desktop/*`），**不改** `internal/agent`/`permission`/`session` 等核心语义。

**非目标（本总纲）**：
- Windows / Linux 打包与平台差异；
- **仅总纲、无实现**（实现从 v0.2.0 根目录 M0 起）；
- 助手 HTML 输出模式的实现（仅列为 M4+ 设计预留，需独立安全 PoC）；
- 改变 `TurnCallbacks`、`permission.Engine`、SQLite schema 的既有语义；
- 云端会话同步、多用户协作、IDE 插件。
## 2. 用户故事

### US-1：同时管理多个项目

**作为** 同时维护多个仓库的开发者，
**我希望** 在一个窗口里通过左侧栏切换不同 workspace（项目），
**以便** 无需为每个项目开新终端 / 新进程。

**验收**：FR-1；AC-1。

### US-2：像原生 App 一样打开项目

**作为** 不熟悉终端的用户，
**我希望** 通过「打开文件夹」或把文件夹拖进窗口来添加 workspace，
**以便** 无需记忆命令行参数即可开始。

**验收**：FR-1.4、FR-16.4；AC-1、AC-9。

### US-3：Workspace 下多个 Agent 对话窗口

**作为** 长期使用者，
**我希望** 在每个 workspace 下能新建多个 Agent 对话窗口、查看列表、恢复任一窗口的历史，
**以便** 在同一项目内并行或切换不同 agent 上下文（对应 TUI 的多 session）。

**验收**：FR-2；AC-2。

### US-4：流畅的流式聊天

**作为** 使用者，
**我希望** 助手回复以 Markdown 流式渲染、代码高亮、thinking 可折叠，且长会话不卡顿，
**以便** 获得优于终端的阅读体验。

**验收**：FR-4；AC-4。

### US-5：不被弹窗打断的权限审批

**作为** 注重节奏的用户，
**我希望** 权限审批（写文件 / shell / web_fetch）以对话流内联卡片出现（允许 / 始终允许 / 拒绝），
**以便** 审批不打断我的视线焦点，且历史可追溯。

**验收**：FR-6；AC-6。

### US-6：可见地停止当前回合

**作为** 使用者，
**我希望** 助手运行时输入区出现明显的「停止」按钮，点击即可取消当前 turn，
**以便** 不依赖 ESC 或记忆快捷键。

**验收**：FR-7；AC-7。

### US-7：图形化查看改动

**作为** review 改动的用户，
**我希望** 右栏检查器能显示 `apply_patch` 的 diff、被读文件预览、工具调用详情，
**以便** 在合并前直观确认。

**验收**：FR-8；AC-8。

### US-8：无弹窗的设置与首启引导

**作为** 首次使用者，
**我希望** 首启在主界面内引导配置 API Key、权限模式、打开首个项目；日常设置在独立视图而非弹窗，
**以便** 配置过程连贯不打断。

**验收**：FR-14、FR-15；AC-10。

### US-9：桌面级系统集成

**作为** macOS 用户，
**我希望** 后台任务完成有系统通知、Dock badge，菜单栏有标准菜单与快捷键，
**以便** 桌面体验符合 macOS 习惯。

**验收**：FR-16；AC-9。

## 3. 功能需求

> 优先级：P0（本版必设计）/ P1（应设计）/ P2（可后置）。「目标期」M1–M4 见 [§7](#7-分期实施路线)。

### FR-0 应用形态与平台

| ID     | 描述                                                                                           | 优先级 | 目标期 |
| ------ | ---------------------------------------------------------------------------------------------- | ------ | ------ |
| FR-0.1 | 新增桌面入口 `cmd/ds-code-desktop`（Wails v3），与 `cmd/ds-code`（CLI/TUI）并列                | P0     | M1     |
| FR-0.2 | 目标平台 **仅 macOS ≥ 12（Monterey）**；单一 universal（arm64 + x86_64）二进制                 | P0     | M1     |
| FR-0.3 | 桌面壳 **Wails v3**；单向流式走 Events，请求-响应走 binding                                     | P0     | M1     |
| FR-0.4 | 复用 `cmd/ds-code/app` 组装逻辑；**不改** `internal/*` 核心语义                                 | P0     | M1     |
| FR-0.5 | CLI/TUI 与桌面 **同 repo、共享 `internal/*`**；TUI 长期保留                                     | P0     | M1     |
| FR-0.6 | 渲染采用**内嵌固定版本浏览器内核**（bundled Chromium），不依赖 macOS 系统 WKWebView 版本       | P0     | M0     |
| FR-0.7 | 前端技术栈：**React + Vite + TypeScript**；UI 组件 shadcn/ui + Tailwind                        | P0     | M1     |
| FR-0.8 | 前端工具链：**Bun ≥ 1.3** 为默认包管理/脚本运行时；Vite 负责 dev/build；Node.js ≥ 26 作 fallback；`package.json` 锁定 `engines` | P0     | M0     |

### FR-1 Workspace 管理

| ID     | 描述                                                                                                    | 优先级 | 目标期 |
| ------ | ----------------------------------------------------------------------------------------------------- | ------ | ------ |
| FR-1.1 | 一个 workspace 绑定一个 `ProjectRoot`；对应一个后端 `app.App` 实例（含独立 Store/MCP/LSP/Checkpoint）  | P0     | M1     |
| FR-1.2 | 左侧栏列出所有已添加 workspace，点击切换活动 workspace                                                 | P0     | M1     |
| FR-1.3 | Workspace 注册表持久化到用户级数据目录（见 [DESIGN §5](DESIGN.md#5-workspace-管理)），跨启动保留       | P0     | M1     |
| FR-1.4 | 添加 workspace：菜单/按钮「打开文件夹」（系统原生选择器）或拖拽文件夹入窗                               | P0     | M1     |
| FR-1.5 | 每个 workspace 展示名称（默认目录名，可重命名）、路径、状态（活动/空闲/运行中）                        | P1     | M1     |
| FR-1.6 | 移除 workspace 仅从注册表移除，**不删除**磁盘项目或 `~/.ds-code/desktop/projects/<project-id>/` 数据 | P0     | M1     |
| FR-1.7 | **沿用** `datadir.ProjectID(ProjectRoot) = hex(SHA256(ProjectRoot))`；**不新增** ID 算法               | P0     | M1     |
| FR-1.8 | 桌面运行时数据目录：`~/.ds-code/desktop/projects/<project-id>/`（sessions.db、checkpoints、audit、shell-jobs、logs）；父目录 `desktop/` 是与 CLI 的唯一区隔点 | P0     | M1     |
| FR-1.9 | 桌面与 CLI/TUI **不共享**上述运行时数据；同一 `project-id` 在 `projects/` 与 `desktop/projects/` 下各自独立库（仍可共用磁盘上的 `.ds-code/config.yaml` 等项目级配置） | P0     | M1     |
| FR-1.10 | 打开 workspace 即懒初始化其 `App`；切换不销毁后台运行中的 workspace                              | P1     | M2     |
| FR-1.11 | 同一 `ProjectRoot` 不可重复添加（去重）；不存在/无权限目录给出内联提示                                  | P1     | M1     |

### FR-2 Agent 对话窗口（对应 TUI session）

| ID     | 描述                                                                                          | 优先级 | 目标期 |
| ------ | --------------------------------------------------------------------------------------------- | ------ | ------ |
| FR-2.1 | 左栏在活动 workspace 下以**树形/分组**列出 Agent 对话窗口（时间倒序，显示标题/时间/模型）        | P0     | M1     |
| FR-2.2 | ⌘N 或侧栏「新建对话」：在当前 workspace 下创建新 Agent 对话窗口（`session.Store.CreateSession`） | P0     | M1     |
| FR-2.3 | 点击某对话窗口 resume：加载该 session 历史消息并渲染（桌面独立 SQLite）                          | P0     | M1     |
| FR-2.4 | 对话窗口重命名 / 删除（删除遵循「消息只增」约束：仅隐藏/标记，不物理删历史行）                  | P1     | M2     |
| FR-2.5 | 对话窗口搜索（按标题/内容）                                                                    | P2     | M3     |
| FR-2.6 | 活动对话窗口状态徽标：运行中 / 等待审批 / 空闲                                                 | P1     | M1     |
| FR-2.7 | 中区顶部显示当前 workspace 名 + 当前 Agent 对话窗口标题                                        | P1     | M1     |

### FR-3 主界面布局（三栏）

| ID     | 描述                                                                                            | 优先级 | 目标期 |
| ------ | ---------------------------------------------------------------------------------------------- | ------ | ------ |
| FR-3.1 | 三栏：左（workspace + Agent 对话窗口导航）/ 中（聊天主区）/ 右（上下文检查器）                  | P0     | M1     |
| FR-3.2 | 左栏与右栏均可折叠（⌘ 快捷键 + 分隔条拖拽）；折叠状态持久化                                      | P0     | M1     |
| FR-3.3 | 右栏（Inspector）默认折叠，按需展开（工具卡片点击 / diff 请求时自动展开）                       | P1     | M2     |
| FR-3.4 | 顶部使用 macOS 原生 titlebar 风格（可 unified/透明），承载 workspace 名称与全局操作            | P1     | M1     |
| FR-3.5 | 底部状态栏：模型、权限模式、token 用量、workspace 状态                                          | P1     | M1     |
| FR-3.6 | 窗口尺寸/栏宽/折叠状态持久化并在下次启动恢复                                                    | P1     | M1     |

### FR-4 聊天与流式渲染

| ID     | 描述                                                                                                     | 优先级 | 目标期 |
| ------ | ------------------------------------------------------------------------------------------------------- | ------ | ------ |
| FR-4.1 | 助手回复默认 **Markdown**；流式阶段轻渲染（pre-wrap + 已闭合代码块高亮），段末全量渲染                   | P0     | M1     |
| FR-4.2 | 复用 [DESIGN §8 Envelope v1](DESIGN.md#8-流式桥接bridge层) 事件协议；TS 两层状态（摄取/渲染）           | P0     | M1     |
| FR-4.3 | thinking / reasoning 独立折叠区，流式追加；首个 content delta 后按规则收起                                | P0     | M1     |
| FR-4.4 | 长会话虚拟列表（仅可见块挂载 DOM）；streaming 块 sticky-follow，用户上滚时暂停跟底                        | P0     | M1     |
| FR-4.5 | 代码块语法高亮 + 复制按钮；行内代码、表格、列表、引用等标准 Markdown 元素                                | P0     | M1     |
| FR-4.6 | 输入区：多行、⌘Enter 发送、`@` 引用文件/路径、`/` 触发 slash 补全                                        | P1     | M1     |
| FR-4.7 | 消息操作：复制、引用回复、（用户消息）编辑重发                                                            | P2     | M2     |
| FR-4.8 | HTML 输出模式为可选、默认关闭（M4+，需安全 PoC）；`content_format` 列可预埋恒为 `markdown`               | P2     | M4     |

### FR-5 工具卡片

| ID     | 描述                                                                                              | 优先级 | 目标期 |
| ------ | ------------------------------------------------------------------------------------------------ | ------ | ------ |
| FR-5.1 | 工具调用渲染为独立卡片（与 assistant 块同级、按序排列），映射 `OnToolStart`/`OnToolEnd`           | P0     | M1     |
| FR-5.2 | 卡片头：工具名 + 摘要参数 + 状态（运行中/成功/失败）+ 耗时                                         | P0     | M1     |
| FR-5.3 | `read`：文件名 + 行范围；`grep`/`glob`：查询 + 命中数；`bash`：命令 + 输出（可折叠）              | P0     | M1     |
| FR-5.4 | `apply_patch`：摘要（改动文件与增删行）；点击在右栏 Inspector 打开 diff                            | P0     | M2     |
| FR-5.5 | 卡片可折叠；大输出默认折叠 + 「展开」；超大结果分片加载                                            | P1     | M1     |
| FR-5.6 | 失败卡片高亮错误信息（`isError`）                                                                 | P0     | M1     |
| FR-5.7 | MCP 工具卡片：以裸名显示，标注来源 server                                                         | P1     | M2     |

### FR-6 权限审批（内联，无弹窗）

| ID     | 描述                                                                                                              | 优先级 | 目标期 |
| ------ | --------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| FR-6.1 | 新增 `DesktopPrompter`（write/shell 二选一）：Go 发事件 → 前端**对话流内联审批卡片** → binding 回传 allow/deny   | P0     | M1     |
| FR-6.2 | 新增 `DesktopWebFetchPrompter`（web_fetch 三选一）：允许一次 / 始终允许 / 拒绝，复用 `WebFetchChoice`             | P0     | M1     |
| FR-6.3 | 审批卡片出现在对话时间线中，含工具名、摘要（文件路径 / 命令 / host+URL）；**不使用打断式弹窗**                    | P0     | M1     |
| FR-6.4 | 「始终允许」（web_fetch）：追加 hostname 到运行时 `Engine.WebAllowlist` 并写项目 `.ds-code/config.yaml`（沿用 v0.1.5） | P0     | M1     |
| FR-6.5 | 审批期间该 turn 阻塞等待；未响应时会话状态显示「等待审批」，可从状态栏/会话徽标感知                               | P0     | M1     |
| FR-6.6 | S3 denylist 与 SSRF 硬规则不受 UI 影响，始终生效（沿用 permission 语义）                                         | P0     | M1     |
| FR-6.7 | 审批结果与历史一并可追溯（作为对话流内的一条记录）                                                               | P1     | M2     |
| FR-6.8 | 键盘可达性：审批卡片可用 Tab/Return 操作，但**不设全局劫持快捷键**                                               | P2     | M2     |

### FR-7 停止 / 取消 turn

| ID     | 描述                                                                                     | 优先级 | 目标期 |
| ------ | --------------------------------------------------------------------------------------- | ------ | ------ |
| FR-7.1 | 助手运行时，输入区的「发送」按钮变为可见**「停止」按钮**；点击取消当前 turn              | P0     | M1     |
| FR-7.2 | 取消经 binding 调用 `context.Cancel`（贯穿子轮次/工具/子代理），复用现有取消链           | P0     | M1     |
| FR-7.3 | **不使用 ESC 作为取消 turn 的动线**（ESC 仅用于关闭浮层/命令面板等局部 UI）              | P0     | M1     |
| FR-7.4 | 取消后：清理流式缓冲、给出「已停止」状态、已产生内容保留                                  | P0     | M1     |
| FR-7.5 | 不设置全局取消快捷键（避免与 macOS 习惯冲突）；如后续加，以 ⌘. 作为可选补充              | P2     | M3     |

### FR-8 上下文检查器（右栏）

| ID     | 描述                                                                                       | 优先级 | 目标期 |
| ------ | ----------------------------------------------------------------------------------------- | ------ | ------ |
| FR-8.1 | 内嵌 Monaco 显示 `apply_patch` diff（并排/内联切换）                                       | P1     | M2     |
| FR-8.2 | 显示被 `read` 的文件预览（只读，语法高亮）                                                 | P2     | M2     |
| FR-8.3 | 显示选中工具卡片的完整参数与结果（大结果分片）                                             | P1     | M2     |
| FR-8.4 | Inspector 内容跟随选中的对话元素；无选中时可显示 workspace 概览                            | P2     | M3     |
| FR-8.5 | diff 只读展示，不提供在 Inspector 内直接编辑写回（编辑仍由 agent `apply_patch` 完成）      | P1     | M2     |

### FR-9 Plan 模式

| ID     | 描述                                                                                     | 优先级 | 目标期 |
| ------ | --------------------------------------------------------------------------------------- | ------ | ------ |
| FR-9.1 | 会话级模式切换（Agent / Plan / 权限模式 readonly/ask/auto），复用 `runmode` 与 permission | P1     | M2     |
| FR-9.2 | Plan 模式下工具集裁剪（read/grep/glob/diagnostics + 可选 web_fetch），UI 明确提示只读     | P1     | M2     |
| FR-9.3 | 模式切换以段控件/下拉呈现（非弹窗）                                                        | P1     | M2     |

### FR-10 子代理面板

| ID      | 描述                                                                                          | 优先级 | 目标期 |
| ------- | -------------------------------------------------------------------------------------------- | ------ | ------ |
| FR-10.1 | 子代理（task 工具）事件按 `streamId=subagent:<id>` 分轨渲染                                    | P1     | M2     |
| FR-10.2 | 同步子代理：主对话流内嵌可展开卡片；后台子代理：侧栏 tab + 完成通知                            | P1     | M2     |
| FR-10.3 | 后台子代理完成时（`OnBackgroundAgentComplete`）发系统通知，点击聚焦对应 tab                    | P1     | M3     |
| FR-10.4 | 子代理工具事件（`OnSubagentTool*`）在其轨内渲染为卡片                                          | P2     | M2     |

### FR-11 slash 命令 / 命令面板

| ID      | 描述                                                                              | 优先级 | 目标期 |
| ------- | -------------------------------------------------------------------------------- | ------ | ------ |
| FR-11.1 | ⌘K 命令面板：搜索并执行全局操作与 slash 命令（复用 `slashcmd`）                    | P1     | M2     |
| FR-11.2 | 输入框 `/` 前缀触发 slash 补全（`/compact`、`/clear`、`/context` 等）             | P1     | M2     |
| FR-11.3 | 命令结果内联到对话流或状态栏，非弹窗                                              | P1     | M2     |
| FR-11.4 | 命令面板亦承载 workspace/会话切换、设置入口等导航                                 | P2     | M3     |

### FR-12 checkpoint / rewind

| ID      | 描述                                                                        | 优先级 | 目标期 |
| ------- | -------------------------------------------------------------------------- | ------ | ------ |
| FR-12.1 | 展示写操作前的 checkpoint 时间线（复用 `internal/checkpoint`）              | P2     | M3     |
| FR-12.2 | rewind 到某 checkpoint：还原文件状态，UI 内联确认（非弹窗）                 | P2     | M3     |
| FR-12.3 | rewind 影响可视化（受影响文件列表 + diff）                                  | P2     | M3     |

### FR-13 MCP / LSP 管理 UI

| ID      | 描述                                                                          | 优先级 | 目标期 |
| ------- | ---------------------------------------------------------------------------- | ------ | ------ |
| FR-13.1 | 设置内展示 MCP servers 状态（已连接/失败/被跳过的工具及原因）                 | P2     | M3     |
| FR-13.2 | 展示 LSP servers 状态（诊断计数、warmup）                                     | P2     | M3     |
| FR-13.3 | 编辑 MCP/LSP 配置（写用户级或项目级 config，原子写）                          | P2     | M3     |
| FR-13.4 | 子进程 PATH/依赖缺失时给出明确内联提示与修复指引                             | P1     | M2     |

### FR-14 设置（视图，非弹窗）

| ID      | 描述                                                                                | 优先级 | 目标期 |
| ------- | ---------------------------------------------------------------------------------- | ------ | ------ |
| FR-14.1 | 设置以**独立视图/侧栏路由**呈现（非弹窗），⌘, 打开                                   | P0     | M1     |
| FR-14.2 | 分区：通用、API Key/模型、权限、外观、MCP、LSP、追踪（tracing）、关于               | P1     | M1     |
| FR-14.3 | 读写 `internal/config`（用户级 `~/.ds-code/config/` + 项目级 `.ds-code/`），原子写 | P0     | M1     |
| FR-14.4 | API Key **沿用 CLI 方式**：仅环境变量 `DS_CODE_DEEPSEEK_API_KEY` / `DEEPSEEK_API_KEY`（`config.LoadAPIKey`）；**禁止**写入 YAML；设置页展示配置状态与指引 | P0     | M1     |
| FR-14.5 | 项目级设置作用于当前 workspace；用户级为默认                                        | P1     | M1     |

### FR-15 Onboarding / 首启引导

| ID      | 描述                                                                             | 优先级 | 目标期 |
| ------- | ------------------------------------------------------------------------------- | ------ | ------ |
| FR-15.1 | 首启在主界面内引导：确认 API Key 已设（`DS_CODE_DEEPSEEK_API_KEY` 等环境变量指引）→ 权限模式 → 打开首个 workspace（内联，非弹窗） | P0     | M1     |
| FR-15.2 | 检测 `git`/`node`/`gopls` 等子进程依赖，缺失时给出安装指引                       | P1     | M2     |
| FR-15.3 | 空状态（无 workspace / 无会话）提供清晰的引导 CTA                                | P0     | M1     |

### FR-16 系统集成（macOS）

| ID      | 描述                                                                             | 优先级 | 目标期 |
| ------- | ------------------------------------------------------------------------------- | ------ | ------ |
| FR-16.1 | 原生菜单栏（App/File/Edit/View/Workspace/Window/Help）+ 标准快捷键              | P1     | M1     |
| FR-16.2 | 系统通知：后台 agent 完成、等待审批（可在设置关闭）                              | P1     | M2     |
| FR-16.3 | Dock badge：运行中/待审批计数                                                    | P2     | M2     |
| FR-16.4 | 拖拽文件夹到窗口 = 添加 workspace；拖拽文件到输入框 = `@` 引用                    | P1     | M2     |
| FR-16.5 | 系统托盘常驻（可选）：快速新建会话、显示运行中任务                                | P2     | M3     |
| FR-16.6 | 深色/浅色模式跟随系统                                                            | P1     | M1     |

### FR-17 后端桥接层（bridge）

| ID      | 描述                                                                                                     | 优先级 | 目标期 |
| ------- | ------------------------------------------------------------------------------------------------------- | ------ | ------ |
| FR-17.1 | 新增 `desktop/bridge`：`TurnCallbacks` → `StreamEmitter` → Wails Events（Envelope v1）                   | P0     | M1     |
| FR-17.2 | Go 侧双层 batch：buffer + flush（16–32ms / maxChunk），边界 flush（tool start / segment end / turn done）| P0     | M1     |
| FR-17.3 | critical 事件（tool/turn/permission）不可丢，带重试；content/reasoning delta 可丢帧                       | P0     | M1     |
| FR-17.4 | binding（请求-响应）：发消息、取消 turn、权限回复、会话/workspace 列表、配置读写                          | P0     | M1     |
| FR-17.5 | `streamId` 分轨（main / subagent:<id>），为多窗口预留                                                    | P1     | M2     |
| FR-17.6 | （建议）抽取 `internal/ui/port`：TUI 与 desktop 共享 StreamBuffer/事件语义与 golden 测试                  | P2     | M2     |

### FR-18 分发与打包（macOS）

| ID      | 描述                                                                                   | 优先级 | 目标期 |
| ------- | ------------------------------------------------------------------------------------- | ------ | ------ |
| FR-18.1 | 打包为 `.app` + `.dmg`；universal 二进制（arm64 + x86_64）                             | P0     | M1     |
| FR-18.2 | Developer ID 签名 + Apple 公证（notarization）+ stapling                               | P0     | M2     |
| FR-18.3 | Hardened Runtime 下保证子进程（git/node/gopls/MCP）可 spawn；PATH 解析策略             | P0     | M2     |
| FR-18.4 | 自动更新（Sparkle 或等价）                                                             | P2     | M3     |
| FR-18.5 | 内嵌 Chromium 使体积大于纯系统 WebView；安装包目标 < 200MB（universal）                         | P1     | M2     |

## 4. 与 CLI/TUI 的行为差异对照

| 维度        | CLI/TUI                              | 桌面（v0.2.0 设计）                              |
| ----------- | ------------------------------------ | ----------------------------------------------- |
| 项目范围    | 单进程绑定一个 `ProjectRoot`         | 单窗口管理多个 workspace                        |
| 取消 turn   | ESC                                  | 可见「停止」按钮（不使用 ESC）                  |
| 权限审批    | TUI overlay / stdin 提示             | 对话流内联审批卡片（无弹窗）                    |
| 设置        | YAML 文件 / slash                    | 独立设置视图（非弹窗）+ 仍读写同一 YAML          |
| slash 命令  | 输入框命令                           | ⌘K 命令面板 + `/` 补全                          |
| diff 查看   | 终端文本                             | 右栏 Monaco 图形化 diff                        |
| 依赖发现    | 报错到终端                           | 内联引导 + onboarding 检测                       |
| 会话 / 数据   | `~/.ds-code/projects/<project-id>/`    | `~/.ds-code/desktop/projects/<project-id>/`；同 id、不同父目录、**不互通** |
| API Key       | 环境变量 `DS_CODE_DEEPSEEK_API_KEY` 等 | **相同**（`config.LoadAPIKey`，禁止 YAML）            |
| 项目级配置    | `<project>/.ds-code/config.yaml`       | **可读同一文件**（allowlist、MCP 等）；各自运行时数据隔离 |

## 5. 非功能需求

| ID     | 描述                                                                                        |
| ------ | ------------------------------------------------------------------------------------------- |
| NFR-1  | 流式阶段 UI 线程 ≥ 55fps（60Hz 屏）；单 turn 跨 Wails 边界调用 < 200 次                       |
| NFR-2  | 冷启动到可交互 < 1.5s（不含 MCP/LSP 子进程 warmup）                                          |
| NFR-3  | 切换 workspace 不阻塞 UI；后台运行中的 workspace 不被中断                                     |
| NFR-4  | 遵循 macOS HIG：动效克制、深浅色适配、可访问性（VoiceOver 基本可用）                          |
| NFR-5  | 后端核心零改动语义；桌面只新增适配层，`make test` 全绿                                        |
| NFR-6  | 桌面运行时数据与 CLI/TUI **隔离**（`ProjectID` 算法相同，仅数据父目录 `desktop/` 不同）；SQLite **schema 相同**、路径不同 |
| NFR-7  | API Key 仅环境变量（与 CLI 一致）；日志不含密钥                                                 |
| NFR-8  | 内嵌 Chromium；安装包 < 200MB（universal）；不依赖系统 WKWebView 版本                          |
| NFR-9  | 权限 S3 denylist / SSRF 硬规则在桌面下同样始终生效                                            |
| NFR-10 | 崩溃/异常不丢失已持久化会话（SQLite 已落盘）；bridge 断连可从 SQLite 兜底最终态               |

## 6. 范围边界

**In scope（设计）**
- `cmd/ds-code-desktop`（Wails v3 入口、菜单、生命周期）
- `desktop/frontend`（TypeScript UI：三栏、聊天、工具卡片、审批卡片、检查器、设置、命令面板）
- `desktop/bridge`（StreamEmitter、Envelope v1、callbacks 适配）
- `desktop/permission`（DesktopPrompter、DesktopWebFetchPrompter）
- Workspace 管理层（多 `app.App` 生命周期 + 注册表持久化）
- `desktop/datadir/`（桌面数据路径 `~/.ds-code/desktop/projects/<project-id>/`；复用 `datadir.ProjectID`）
- 系统集成（菜单栏、通知、Dock、拖拽、深浅色）
- 打包/签名/公证/PATH/onboarding
- `docs/v0.2.0/**`（含 `spec/` 总纲与本目录实现文档）
- `docs/v0.2.1/**` … `docs/v0.2.4/**`

**Out of scope**
- Windows / Linux 平台
- 实现代码交付（本版仅文档）
- 助手 HTML 输出模式实现（M4+，仅设计预留）
- 改动 `internal/agent`/`permission`/`session` 核心语义与 SQLite schema（除 `content_format` 预埋列，恒 markdown）
- 云端同步、协作、IDE 插件
- 在 Inspector 内直接编辑并写回文件（编辑仍由 agent `apply_patch`）

## 7. 分期实施路线

> 完整 FR 见上文；下表映射到 **实现版本**（v0.2.0 根目录起即为可交付代码）。路线图：[ROADMAP.md](ROADMAP.md)。

| 里程碑 | 实现版本 | 目标 | gate |
| ------ | -------- | ---- | ---- |
| **M0** | [v0.2.0 根](../) | Wails + Chromium + bridge + 单 workspace 流式/审批/停止 | 能打开项目、发消息、看流式、批准一次写 |
| **M1** | [v0.2.1](../../v0.2.1/) | 多 workspace + 三栏 + P0 + 打包 | FR-0/1/2/3/4/5/6/7/14/15/17（P0）绿 |
| **M2** | [v0.2.2](../../v0.2.2/) | Inspector + Plan + 子代理 + 命令面板 + 公证 | FR-8/9/10/11/13.4/16/18.2/18.3 |
| **M3** | [v0.2.3](../../v0.2.3/) | checkpoint + MCP/LSP + 托盘 + 更新 | FR-12/13/16.5/2.5/18.4 |
| **M4** | [v0.2.4](../../v0.2.4/) | HTML 输出（可选） | FR-4.8；OWASP XSS 全通过 |
