# v0.2.0 phase0 验收文档

> 版本：v0.2.0
> 阶段：phase0（M0 PoC）
> 状态：待实现
> 总纲：[spec/ACCEPTANCE](../spec/ACCEPTANCE.md)
> 更新日期：2026-07-07

## Gate 总述

本阶段通过即表示技术栈与桥接契约可行，可进入 phase1 MVP 开发。

## 必过验收

### AC-11 bridge / 协议

| ID | 标准 |
| -- | ---- |
| AC-11.1 | StreamEmitter 单测：delta 序列 emit 次数/batch/flush 符合预期 |
| AC-11.2 | golden JSON：一次完整 turn 的 Envelope v1 序列稳定 |
| AC-11.3 | critical 不丢；content/reasoning 背压可丢帧 |

### AC-6 权限（最小）

| ID | 标准 |
| -- | ---- |
| AC-6.1 | write/shell 内联审批卡片，无弹窗 |
| AC-6.5 | S3 denylist 命中始终拒绝 |

### AC-7 停止

| ID | 标准 |
| -- | ---- |
| AC-7.1 | 「停止」按钮数秒内取消 turn |
| AC-7.2 | ESC 不取消 turn |

### AC-12 渲染引擎

| ID | 标准 |
| -- | ---- |
| AC-12.5 | 内嵌 Chromium；不默认回退 WKWebView |

### AC-13 工具链

| ID | 标准 |
| -- | ---- |
| AC-13.1 | `PACKAGE_MANAGER=bun wails3 dev` HMR 正常 |
| AC-13.2 | shadcn button 可渲染 |
| AC-13.3 | `wails3 build` 产出 `.app` |
| AC-13.4 | `bun run test` Vitest 可跑 |
| AC-13.5 | `engines.bun >= 1.3.0`、`engines.node >= 26.0.0` |

### AC-4 流式（子集）

| ID | 标准 |
| -- | ---- |
| AC-4.3 | 长回复流式 ≥ 55fps |
| AC-4.5 | 单 turn 跨边界 < 200 次 |

## 回归

| ID | 标准 |
| -- | ---- |
| REG-1 | `make test` / `make lint` / `make vet` 全绿 |
| REG-3 | 数据写入 `~/.ds-code/desktop/projects/<project-id>/` |

## 手动验证步骤

1. 指定一个含 `.ds-code/` 的项目目录启动桌面 App
2. 发送消息，观察流式回复
3. ask 模式下触发写文件，出现内联审批 → 允许
4. 长任务运行中点击「停止」，确认取消且 ESC 无效
5. 检查 `~/.ds-code/desktop/projects/<id>/sessions.db` 已创建
6. 运行 `bun run test`、bridge 单测

## 明确不要求（本阶段）

- AC-1 多 workspace、AC-2 多 session 列表、AC-3 三栏
- AC-6.2 web_fetch 三选一、AC-10 设置/onboarding
- AC-12.1/12.2 分发公证
