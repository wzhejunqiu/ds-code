# v0.2.1 设计文档

> 版本：v0.2.1（M1 MVP）
> 状态：待实现
> 前置：[v0.2.0](../v0.2.0/)　总纲：[spec/DESIGN](../v0.2.0/spec/DESIGN.md)
> 更新日期：2026-07-07

## 1. 本版定位

在 v0.2.0 单 workspace PoC 上扩展为完整 MVP。**复用** v0.2.0 的 bridge/datadir/permission，新增 workspace 层与三栏前端。

## 2. 新增模块

```
desktop/workspace/         # Manager、registry、session_facade
desktop/frontend/src/      # 三栏、侧栏、工具卡片、设置
cmd/ds-code-desktop/       # menu.go、lifecycle.go
```

## 3. 实现焦点（引用总纲 spec/）

| 主题 | 总纲章节 | 本版要点 |
| ---- | -------- | -------- |
| WorkspaceManager | §4.1、§5 | 多 App 懒初始化、`workspaces.json` |
| 三栏 UI | §6 | Inspector **占位折叠** |
| DesktopWebFetchPrompter | §9.2 | web_fetch 三选一 |
| 设置/onboarding | §6.1.7–6.1.8 | `#/settings`、首启空状态 |
| 打包 | §15 | universal `.app` + `.dmg`（无公证） |

## 4. 与 v0.2.0 的兼容

- Envelope v1 不变；`workspaceId` 从 `"default"` 改为真实 ID
- bridge/datadir 契约不重构
