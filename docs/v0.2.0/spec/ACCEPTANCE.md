# v0.2.0 验收文档

> 版本：v0.2.0
> 状态：设计中（Design）
> 更新日期：2026-07-04
> 需求：[REQUIREMENTS.md](REQUIREMENTS.md)　设计：[DESIGN.md](DESIGN.md)

## 说明

本目录为**设计总纲**（`spec/`）。下列验收标准分三类：

- **DOC 验收**：`spec/` 文档完整性、决策一致性（本目录可核验）。
- **IMPL 验收（AC-x）**：总纲级功能标准；**各 phase** scoped 清单见 [phase0](../phase0/ACCEPTANCE.md)、[phase1](../phase1/ACCEPTANCE.md) 等。
- **阶段 gate**：见 [ROADMAP.md](ROADMAP.md)。

## A. 文档验收（本版必须通过）

| ID     | 标准                                                                                              |
| ------ | ----------------------------------------------------------------------------------------------- |
| DOC-1  | `docs/v0.2.0/spec/` 含总纲四件套 + ROADMAP；`docs/v0.2.0/phase0/`~`phase4/` 各有实现版四件套 |
| DOC-2  | 六项核心立场（macOS 唯一、UI 重设计、无弹窗、workspace、三栏、Wails v3）在三份文档中口径一致       |
| DOC-3  | 所有 FR 均有目标里程碑与优先级；FR ↔ US ↔ AC 可追溯                                               |
| DOC-4  | 明确「不改 `TurnCallbacks`/`permission.Engine`/`session.Store`/SQLite schema 语义」               |
| DOC-5  | `docs/README.md` v0.2.0 行更新；[DESKTOP.md](../../DESKTOP.md) 指向 `v0.2.0/spec/` |
| DOC-6  | `ProjectID` 算法不变、桌面数据靠 `~/.ds-code/desktop/projects/<project-id>/` 目录隔离被明确写明     |
| DOC-7  | 无弹窗策略对「权限/设置/命令/取消」四处均给出具体替代方案                                          |
| DOC-8  | workspace 下 Agent 对话窗口（= TUI session）层级与 UI 导航被明确写明                               |
| DOC-9  | 内嵌 Chromium 渲染策略、React+Vite+TS+Bun 工具链已定稿、API Key 沿用 env 方式已写明                |

## B. 功能验收（面向实现里程碑）

### AC-1 Workspace 管理（M1，FR-1）

| ID     | 验收标准                                                                              |
| ------ | ------------------------------------------------------------------------------------ |
| AC-1.1 | 「打开文件夹」或拖拽文件夹可添加 workspace，出现在左栏                                 |
| AC-1.2 | 重启后 workspace 列表与活动 workspace 从 `workspaces.json` 恢复                        |
| AC-1.3 | 同一路径重复添加被去重                                                                |
| AC-1.4 | 移除 workspace 后，磁盘项目与 `~/.ds-code/projects/<id>` 数据仍在                     |
| AC-1.5 | 同一路径 project 在 CLI 创建的 session，桌面 workspace **不可见**（同 `project-id`，但数据在 `desktop/projects/` 隔离） |
| AC-1.6 | 切换 workspace 时，另一 workspace 运行中的 turn 不被中断（NFR-3）                     |

**手动步骤**：打开两个仓库 → 各发一条消息 → 切换 → 验证互不干扰 → 重启验证恢复。

### AC-2 Agent 对话窗口（M1，FR-2）

| ID     | 验收标准                                                                 |
| ------ | ------------------------------------------------------------------------ |
| AC-2.1 | 活动 workspace 下以树形/分组列出 Agent 对话窗口（标题/时间/模型）          |
| AC-2.2 | ⌘N 或侧栏「新建对话」在当前 workspace 创建新窗口，写入桌面 `sessions.db`   |
| AC-2.3 | 选中某对话窗口 resume，正确渲染历史消息与工具卡片                          |
| AC-2.4 | 对话窗口徽标反映运行中 / 等待审批 / 空闲                                 |
| AC-2.5 | 中区顶部显示当前 workspace 名 + 当前对话窗口标题                         |

### AC-3 三栏布局（M1，FR-3）

| ID     | 验收标准                                                             |
| ------ | ------------------------------------------------------------------ |
| AC-3.1 | 左/右栏可折叠（⌘\\ / ⌘⌥\\），折叠与栏宽重启后恢复                    |
| AC-3.2 | 右栏 Inspector 默认折叠，点击工具卡片/请求 diff 时自动展开          |
| AC-3.3 | 底部状态栏显示模型 / 权限模式 / token 用量 / workspace 状态         |
| AC-3.4 | 深浅色跟随系统切换即时生效                                          |

### AC-4 聊天与流式（M1，FR-4）

| ID     | 验收标准                                                                                  |
| ------ | ---------------------------------------------------------------------------------------- |
| AC-4.1 | 助手回复流式出现，段末全量 Markdown 渲染（代码高亮/表格/列表正确）                        |
| AC-4.2 | reasoning 折叠区流式追加，首个 content 后按规则收起                                       |
| AC-4.3 | 10k token 长回复流式期 UI ≥ 55fps（60Hz），无卡顿（NFR-1）                                |
| AC-4.4 | 长会话虚拟列表滚动流畅；streaming 块 sticky-follow，用户上滚时暂停跟底                    |
| AC-4.5 | 单 turn 跨 Wails 边界调用 < 200 次（NFR-1）                                              |
| AC-4.6 | ⌘Enter 发送、`@` 引用文件、`/` 触发补全可用                                              |

### AC-5 工具卡片（M1，FR-5）

| ID     | 验收标准                                                                    |
| ------ | ------------------------------------------------------------------------- |
| AC-5.1 | read/grep/glob/bash/apply_patch 均渲染为卡片，头含名/摘要/状态/耗时         |
| AC-5.2 | 大输出默认折叠可展开；失败卡片高亮错误                                     |
| AC-5.3 | `apply_patch` 卡片点击后在右栏打开 diff（M2）                              |
| AC-5.4 | MCP 工具卡片以裸名显示并标注来源 server（M2）                             |

### AC-6 权限内联审批（M1，FR-6）

| ID     | 验收标准                                                                                       |
| ------ | -------------------------------------------------------------------------------------------- |
| AC-6.1 | write/shell 审批以**对话流内联卡片**出现，**无打断式弹窗**                                     |
| AC-6.2 | web_fetch 三选一（允许一次/始终允许/拒绝）可用                                                |
| AC-6.3 | 「始终允许」写入项目 `.ds-code/config.yaml` 且运行时 `WebAllowlist` 生效（对齐 v0.1.5）        |
| AC-6.4 | 等待审批期间会话徽标显示「等待审批」，turn 阻塞不推进                                          |
| AC-6.5 | 拒绝后该工具调用失败并可见原因；S3 denylist / SSRF 命中始终拒绝（不受 UI 影响）                |
| AC-6.6 | 审批记录保留在对话历史中可回溯                                                                |

**手动步骤**：ask 模式让 agent 写文件 → 出现内联审批卡片 → 分别验证允许/拒绝 → 让 agent 访问未列入主机 → 验证三选一与写 config。

### AC-7 停止 turn（M1，FR-7）

| ID     | 验收标准                                                              |
| ------ | ------------------------------------------------------------------- |
| AC-7.1 | 运行时发送按钮变为「停止」按钮；点击后 turn 在数秒内停止              |
| AC-7.2 | **ESC 不取消 turn**（仅关闭浮层/命令面板）                           |
| AC-7.3 | 停止后已产生内容保留，状态显示「已停止」，可继续发新消息             |
| AC-7.4 | 取消贯穿子轮次/工具/子代理（复用 agent 取消链，触发 HookStop）       |

### AC-8 Inspector / diff（M2，FR-8）

| ID     | 验收标准                                                          |
| ------ | -------------------------------------------------------------- |
| AC-8.1 | `apply_patch` 的 diff 在右栏 Monaco 正确显示（并排/内联切换）    |
| AC-8.2 | 被 read 文件在右栏只读预览、语法高亮                             |
| AC-8.3 | 选中工具卡片，Inspector 展示完整参数与结果（大结果分片）         |

### AC-9 系统集成（M1–M2，FR-16）

| ID     | 验收标准                                                          |
| ------ | -------------------------------------------------------------- |
| AC-9.1 | 原生菜单栏与标准快捷键可用                                       |
| AC-9.2 | 后台 agent 完成触发系统通知，点击聚焦对应会话/tab（M2）          |
| AC-9.3 | Dock badge 反映运行中/待审批（M2）                              |
| AC-9.4 | 拖拽文件夹添加 workspace、拖拽文件到输入框成 `@` 引用（M2）      |

### AC-10 设置与 onboarding（M1，FR-14/15）

| ID      | 验收标准                                                                     |
| ------- | -------------------------------------------------------------------------- |
| AC-10.1 | 设置为**独立视图**（非弹窗），⌘, 打开                                        |
| AC-10.2 | API Key 通过 `DS_CODE_DEEPSEEK_API_KEY` / `DEEPSEEK_API_KEY` 加载（`config.LoadAPIKey`）；YAML 无 `llm.api_key`；日志无明文 |
| AC-10.3 | 首启空状态引导配置 API Key（环境变量指引）、权限模式、打开首个 workspace（内联）            |
| AC-10.4 | 设置读写 `internal/config`（用户级 + 项目级）；项目级 `.ds-code/config.yaml` 可与 CLI 共用 |

### AC-13 前端工具链（M0，FR-0.8）

| ID      | 验收标准                                                                                  |
| ------- | --------------------------------------------------------------------------------------- |
| AC-13.1 | `PACKAGE_MANAGER=bun wails3 dev` 可启动，Vite HMR 正常                                   |
| AC-13.2 | `bunx --bun shadcn@latest add button` 后组件可渲染                                        |
| AC-13.3 | `wails3 build` 产出含前端 assets 的 `.app`                                                |
| AC-13.4 | `bun run test`（Vitest reducer 快照）可跑                                                 |
| AC-13.5 | `desktop/frontend/package.json` 含 `engines.bun >= 1.3.0` 与 `engines.node >= 26.0.0`   |

**手动步骤**：在 `desktop/frontend/` 执行 `bun install` → 确认 `package.json` engines → `wails3 dev` 验证 HMR → `wails3 build` 验证打包 → `bun run test` 验证单测。

### AC-11 bridge / 协议（M0–M1，FR-17）

| ID      | 验收标准                                                                          |
| ------- | ------------------------------------------------------------------------------- |
| AC-11.1 | StreamEmitter 单测：给定 delta 序列，emit 次数/batch 大小/flush 边界符合预期      |
| AC-11.2 | golden JSON：一次完整 turn 的 Envelope v1 序列稳定                                |
| AC-11.3 | critical 事件（tool/turn/permission）不丢；content/reasoning 背压下可丢帧         |
| AC-11.4 | `turn.done` 后可从 SQLite 拉最终态与流式累积一致（NFR-10）                        |
| AC-11.5 | 多 workspace 下事件按 `workspaceId` 正确路由，后台 workspace 不误渲染到中区       |

### AC-12 分发打包（M1–M2，FR-18）

| ID      | 验收标准                                                                  |
| ------- | ---------------------------------------------------------------------- |
| AC-12.1 | 产出 universal `.app` + `.dmg`，在 macOS 12+ 可运行                     |
| AC-12.2 | 签名 + 公证 + staple 通过；Gatekeeper 不报警（M2）                      |
| AC-12.3 | Hardened Runtime 下 git/node/gopls/MCP 子进程可正常 spawn（M2）         |
| AC-12.4 | 安装包 < 200MB（内嵌 Chromium + universal，NFR-8）                         |
| AC-12.5 | 使用内嵌 Chromium 渲染；不默认回退系统 WKWebView（M0 PoC 门禁）            |

## C. 回归与约束

| ID     | 标准                                                                            |
| ------ | ------------------------------------------------------------------------------ |
| REG-1  | `make test` / `make lint` / `make vet` 全绿；后端核心无语义改动（NFR-5）         |
| REG-2  | CLI/TUI 功能不受桌面新增代码影响                                                 |
| REG-3  | 桌面运行时数据写入 `~/.ds-code/desktop/projects/<project-id>/`；与 CLI `~/.ds-code/projects/<project-id>/` 同 id、不同父目录、**隔离**（NFR-6） |
| REG-4  | 权限 S3 denylist / SSRF 硬规则在桌面同样生效（NFR-9）                            |

## D. 里程碑门禁总表

| 里程碑 | Phase | 必过验收 |
| ------ | ----- | -------- |
| M0 | [phase0](../phase0/ACCEPTANCE.md) | AC-11.1/11.2/11.3、AC-7.1、AC-6.1、AC-12.5、AC-13 |
| M1 | [phase1](../phase1/ACCEPTANCE.md) | AC-1~5、AC-6、AC-7、AC-10、AC-11、REG-1~4、AC-12.1/12.4 |
| M2 | [phase2](../phase2/ACCEPTANCE.md) | AC-8、AC-9、AC-12.2/12.3、Plan/子代理/命令面板 |
| M3 | [phase3](../phase3/ACCEPTANCE.md) | checkpoint、MCP/LSP、billing、托盘、自动更新 |
| M4 | [phase4](../phase4/ACCEPTANCE.md) | HTML 输出（OWASP XSS 全通过） |
