# v0.2.0 实现路线图

> 总纲：[spec/README.md](README.md)　需求：[REQUIREMENTS.md](REQUIREMENTS.md)　设计：[DESIGN.md](DESIGN.md)
> 更新日期：2026-07-07

## 文档结构

```
docs/v0.2.0/
  README.md                                              ← v0.2.0 版本总入口
  spec/
    README.md / REQUIREMENTS.md / DESIGN.md / ACCEPTANCE.md / ROADMAP.md   ← 完整总纲
  phase0/ … phase4/                                      ← 实现阶段（多提交拆分）
    README.md / REQUIREMENTS.md / DESIGN.md / ACCEPTANCE.md
```

- **spec/**：完整产品设计总纲（原文保留，供全系列引用）
- **phase0~phase4**：v0.2.0 内的实现阶段；**phase 对应 git 提交拆分，不对应独立 semver**
- 对外发布版本统一为 **v0.2.0**

## 拆分原则

1. **纵向依赖**：后一阶段建立在上一阶段代码与验收之上
2. **横向独立**：同阶段内模块尽量可并行
3. **可交付**：每阶段有明确 gate
4. **追溯**：各 phase FR/AC 引用 spec/ 总纲编号

## Phase 映射

| Phase | 里程碑 | 主题 | Gate |
| ----- | ------ | ---- | ---- |
| [phase0](../phase0/) | M0 | 技术验证 PoC | 单 workspace：流式 + 审批 + 停止 |
| [phase1](../phase1/) | M1 | 桌面 MVP | 多 workspace + 三栏 + P0 + 基础打包 |
| [phase2](../phase2/) | M2 | 对齐增强 | Inspector + Plan + 子代理 + 公证 |
| [phase3](../phase3/) | M3 | 深化 | checkpoint + MCP/LSP + 托盘 + 更新 |
| [phase4](../phase4/) | M4 | HTML（可选） | 安全 PoC 通过后启用 |

```mermaid
flowchart LR
  spec["spec/ 总纲"]
  p0["phase0 M0"]
  p1["phase1 M1"]
  p2["phase2 M2"]
  p3["phase3 M3"]
  p4["phase4 M4"]
  release["v0.2.0 发布"]
  spec -.->|指导| p0
  p0 --> p1 --> p2 --> p3 --> p4
  p4 --> release
```

## 各阶段范围速览

### phase0 — M0

- Wails v3 + 内嵌 Chromium PoC + Bun/Vite 工具链
- `desktop/bridge`、`desktop/datadir`、`desktop/permission`
- 单 workspace 最小聊天 UI

### phase1 — M1

- `WorkspaceManager` + 三栏 React UI + 完整 P0

### phase2 — M2

- Inspector Monaco diff + Plan + 子代理 + ⌘K + 签名公证

### phase3 — M3

- checkpoint/rewind + MCP/LSP UI + 托盘 + Sparkle

### phase4 — M4（可选）

- HTML 输出 + DOMPurify 安全栈

## 与 CLI 版本号

桌面 **v0.2.0** 与 CLI **v0.1.x** 可并行；共享 `internal/*`。发布 `.app` 建议 app 版本与文档 v0.2.0 对齐。
