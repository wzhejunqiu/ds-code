# v0.2.0 需求文档

> 版本：v0.2.0（M0 PoC）
> 状态：待实现
> 总纲：[spec/REQUIREMENTS](spec/REQUIREMENTS.md)
> 更新日期：2026-07-07

## 1. 目标

验证桌面技术栈与后端桥接契约可行，在**单 workspace** 下跑通最小 agent 闭环。

## 2. 本版功能需求

> FR 编号引用总纲，不新增 ID。

### FR-0 应用形态（子集）

| ID | 描述 | 总纲 |
| -- | ---- | ---- |
| FR-0.1 | `cmd/ds-code-desktop` Wails v3 入口可启动 | FR-0.1 |
| FR-0.3 | Events 流式 + binding 请求-响应 | FR-0.3 |
| FR-0.4 | 复用 `app.New` / `newRunner`，不改 `internal/*` 语义 | FR-0.4 |
| FR-0.6 | 内嵌固定版本 Chromium，不默认 WKWebView | FR-0.6 |
| FR-0.7 | React + Vite + TypeScript 前端 | FR-0.7 |
| FR-0.8 | Bun ≥ 1.3 工具链 PoC | FR-0.8 |

### FR-1 Workspace（最小）

| ID | 描述 | 总纲 |
| -- | ---- | ---- |
| FR-1.1 | 单 `ProjectRoot` 绑定一个 `app.App`（启动参数或硬编码） | FR-1.1 |
| FR-1.7 | `datadir.ProjectID` 算法不变 | FR-1.7 |
| FR-1.8 | 数据落 `~/.ds-code/desktop/projects/<project-id>/` | FR-1.8 |

> 多 workspace、注册表、拖拽添加 → **v0.2.1**

### FR-4 聊天与流式（最小）

| ID | 描述 | 总纲 |
| -- | ---- | ---- |
| FR-4.1 | 助手回复流式 pre-wrap；段末 Markdown（可简化） | FR-4.1 |
| FR-4.2 | Envelope v1 + TS reducer 摄取 | FR-4.2 |

> 虚拟列表、reasoning 折叠、代码高亮完整度 → **v0.2.1**

### FR-6 权限审批（最小）

| ID | 描述 | 总纲 |
| -- | ---- | ---- |
| FR-6.1 | `DesktopPrompter` write/shell 二选一，内联卡片 | FR-6.1 |
| FR-6.3 | 无打断式弹窗 | FR-6.3 |
| FR-6.6 | S3 denylist / SSRF 硬规则生效 | FR-6.6 |

> web_fetch 三选一 → **v0.2.1**

### FR-7 停止 turn

| ID | 描述 | 总纲 |
| -- | ---- | ---- |
| FR-7.1 | 运行时「停止」按钮取消 turn | FR-7.1 |
| FR-7.2 | `context.Cancel` 贯穿取消链 | FR-7.2 |
| FR-7.3 | ESC 不取消 turn | FR-7.3 |
| FR-7.4 | 停止后已产生内容保留 | FR-7.4 |

### FR-17 Bridge

| ID | 描述 | 总纲 |
| -- | ---- | ---- |
| FR-17.1 | `TurnCallbacks` → `StreamEmitter` → Events | FR-17.1 |
| FR-17.2 | Go 侧 buffer + flush（16ms / maxChunk） | FR-17.2 |
| FR-17.3 | critical 事件不丢；delta 可丢帧 | FR-17.3 |
| FR-17.4 | binding：发消息、取消、权限回复 | FR-17.4 |

## 3. 非功能需求（本版）

| ID | 描述 |
| -- | ---- |
| NFR-1 | 流式 ≥ 55fps；单 turn 跨边界 < 200 次调用 |
| NFR-5 | `make test` 全绿；后端核心无语义改动 |
| NFR-6 | 桌面数据目录 `desktop/projects/` 隔离 |
| NFR-8 | 内嵌 Chromium PoC 达标 |

## 4. 明确不包含

- 多 workspace、`workspaces.json`（v0.2.1）
- 三栏布局、Inspector、设置、onboarding（v0.2.1）
- 完整工具卡片、Plan、子代理、命令面板（v0.2.1+）
- 签名公证、`.dmg` 分发（v0.2.1 基础打包 / v0.2.2 公证）
- web_fetch 三选一 Prompter（v0.2.1）

## 5. Gate

打开项目 → 发消息 → 流式可见 → 批准一次写 → 停止按钮可取消。
