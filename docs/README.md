# ds-code 文档

本目录按版本组织产品、设计与配置文档。

## 版本文档

| 版本 | 说明 |
|------|------|
| [v0.1.0/](v0.1.0/) | v0.1.0 基线：CONFIG、DESIGN、PLAN、SECURITY、LLM、TUI 测试等 |
| [v0.1.1/](v0.1.1/) | v0.1.1 增量：MCP 裸名注册、TUI header 消息通知区 |
| [v0.1.2/](v0.1.2/) | v0.1.2：路径权限、MCP spill、MCP 参数、搜索不读 gitignore（`.git`+`skip_dirs`） |
| [v0.1.3/](v0.1.3/) | v0.1.3：Bubble Tea v2（`charm.land`）迁移、虚拟列表、选区增强、FR-3/4 延期项 |
| [v0.1.4/](v0.1.4/) | v0.1.4：`shell`→`bash`、grep/glob ripgrep、8 工具 FR-0 prompt；`diagnostics`/`tool_search`/`web_search` prompt 延后 |
| [v0.1.5/](v0.1.5/) | v0.1.5：`web_fetch` allowlist 迁入 permission、三选一审批、「始终允许」写项目 config；OTel 日志 `trace_id`/`span_id` |
| [v0.2.0/](v0.2.0/) | v0.2.0：桌面产品线 — 总纲 [spec/](v0.2.0/spec/)；实现按 [phase0~phase4](v0.2.0/README.md) 多提交交付（M0 PoC → M4 HTML 可选） |

## 演进方向

| 文档 | 说明 |
|------|------|
| [DESKTOP.md](DESKTOP.md) | **（已弃用）** 早期可行性研究；路线已被 [v0.2.0/spec/](v0.2.0/spec/) 取代 |

## 快速入口

- 配置：[v0.1.0/CONFIG.md](v0.1.0/CONFIG.md)
- 系统设计：[v0.1.0/DESIGN.md](v0.1.0/DESIGN.md)
- 安全：[v0.1.0/SECURITY.md](v0.1.0/SECURITY.md)
- 桌面总纲：[v0.2.0/spec/README.md](v0.2.0/spec/README.md)
- 桌面实现：[v0.2.0/README.md](v0.2.0/README.md)（phase0~phase4）
