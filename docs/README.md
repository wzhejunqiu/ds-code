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
| [v0.2.0/](v0.2.0/) | v0.2.0（M0）：桌面 **PoC 实现** — Wails + Chromium + bridge + 单 workspace 流式/审批/停止；总纲见 [spec/](v0.2.0/spec/) |
| [v0.2.1/](v0.2.1/) | v0.2.1（M1）：桌面 MVP — 多 workspace、三栏、工具卡片、设置、onboarding、基础打包 |
| [v0.2.2/](v0.2.2/) | v0.2.2（M2）：Inspector diff、Plan、子代理、命令面板、签名公证 |
| [v0.2.3/](v0.2.3/) | v0.2.3（M3）：checkpoint/rewind、MCP/LSP 管理、托盘、自动更新 |
| [v0.2.4/](v0.2.4/) | v0.2.4（M4，可选）：助手 HTML 输出（安全 PoC 门禁后） |

## 演进方向

| 文档 | 说明 |
|------|------|
| [DESKTOP.md](DESKTOP.md) | **（已弃用）** 早期可行性研究；路线已被 [v0.2.0/spec/](v0.2.0/spec/) 取代 |

## 快速入口

- 配置：[v0.1.0/CONFIG.md](v0.1.0/CONFIG.md)
- 系统设计：[v0.1.0/DESIGN.md](v0.1.0/DESIGN.md)
- 安全：[v0.1.0/SECURITY.md](v0.1.0/SECURITY.md)
- 桌面总纲：[v0.2.0/spec/README.md](v0.2.0/spec/README.md)
- 桌面实现（M0）：[v0.2.0/README.md](v0.2.0/README.md)
