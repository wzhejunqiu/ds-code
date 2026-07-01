# v0.1.5 验收标准

> 版本：v0.1.5
> 状态：已实现
> 更新日期：2026-07-01
> 需求：[REQUIREMENTS.md](REQUIREMENTS.md) · 设计：[DESIGN.md](DESIGN.md)

## 1. 总体验收

- [ ] 版本号标记为 v0.1.5（发布打 `v0.1.5` tag 时由 ldflags 注入；CHANGELOG 已切版）
- [x] `make test` / `make lint` / `make vet` 通过
- [x] `CHANGELOG.md` 含 allowlist 语义 **breaking** 说明
- [x] `configs/example.yaml` `web.allowlist` 注释已更新
- [x] `internal/tool/builtin/web_fetch/web_fetch_policy.go` 已删除
- [x] `configs/example.yaml` 含 `tracing` 段说明
- [x] `CHANGELOG.md` 含 tracing 能力说明（默认关闭；支持 CLI 与 YAML）

**自动化冒烟**：

```bash
go test -race -count=1 ./internal/permission/... ./internal/config/... ./internal/tool/builtin/web_fetch/... ./internal/ui/tui/...
go test -race -count=1 ./internal/trace/... ./internal/logging/... ./internal/agent/... -run 'Trace|Logctx|TraceCore|Web'
make test
```

## 2. allowlist 语义验收（FR-0）

### AC-2.1 readonly/ask + allowlist 命中

| 检查                                | 预期                                         |
| ----------------------------------- | -------------------------------------------- |
| `web.allowlist: [example.com]`      | 访问 `https://example.com/...` **无** prompt |
| 通配 `*.github.io`                  | `foo.github.io` 命中                         |
| SSRF 主机（localhost、169.254.x.x） | **拒绝**，即使在 allowlist                   |

### AC-2.2 auto 模式

| 检查                    | 预期            |
| ----------------------- | --------------- |
| 空 allowlist + 公网 URL | SSRF 通过即放行 |
| 私有 IP / loopback      | 拒绝            |

### AC-2.3 空 allowlist 行为变更（breaking）

| 检查                              | v0.1.4 | v0.1.5            |
| --------------------------------- | ------ | ----------------- |
| readonly + `allowlist: []` + 公网 | 拒绝   | 三选一 prompt     |
| auto + `allowlist: []` + 公网     | 拒绝   | 放行（SSRF 通过） |

## 3. 三选一 Prompter 验收（FR-2、FR-4）

### AC-3.1 TUI 交互

| 检查                     | 预期                                       |
| ------------------------ | ------------------------------------------ |
| 未列入主机触发 web_fetch | 浮层显示 host / URL                        |
| 按 `1` 或 `a`            | 本次请求完成；同 host 重定向不二次 prompt  |
| 按 `2` 或 `s`            | 本次完成；`.ds-code/config.yaml` 含新 host |
| 按 `3` 或 `d`            | `ErrRejected`；工具返回拒绝                |
| write/shell prompt       | 仍为 y/n 二选一（未回归）                  |

### AC-3.2 非 TUI 降级

| 检查                              | 预期                      |
| --------------------------------- | ------------------------- |
| `ds-code -p` + 未列入主机         | `ErrNeedTTY`              |
| stdin 模式（ask + 有 TTY 无 TUI） | 打印三选项；读 stdin 生效 |

### AC-3.3 AllowAlways 内存生效

| 检查                           | 预期                         |
| ------------------------------ | ---------------------------- |
| 选「始终允许」后同会话再次访问 | **无** prompt                |
| 重启进程后                     | config 已合并，**无** prompt |

## 4. config 持久化验收（FR-3）

### AC-4.1 文件写入

| 检查          | 预期                                     |
| ------------- | ---------------------------------------- |
| 目标路径      | `<project>/.ds-code/config.yaml`         |
| 权限          | `0600`                                   |
| 去重          | 重复「始终允许」不重复条目               |
| 保留          | 已有 allowlist 项不丢失                  |
| 用户级 config | `~/.ds-code/config/config.yaml` **不变** |

### AC-4.2 自动化

`go test ./internal/config/... -run WebAllowlist` 绿。

## 5. Engine 与子代理验收（FR-1、FR-5）

### AC-5.1 `Engine.Check(web_fetch)`

| 检查                                         | 预期                            |
| -------------------------------------------- | ------------------------------- |
| `Check("web_fetch", {"url": "https://..."})` | 走 `CheckWebFetch`              |
| deny 日志                                    | `classifyDeny` 可区分 web_fetch |

### AC-5.2 启动注入

| 检查                       | 预期                           |
| -------------------------- | ------------------------------ |
| `runner.Perm.WebAllowlist` | 等于合并后 `cfg.Web.Allowlist` |
| TUI                        | `WebFetchPrompter` 已挂载      |

### AC-5.3 子代理

| 检查                            | 预期                                         |
| ------------------------------- | -------------------------------------------- |
| bubble/inherit 子代理 web_fetch | 可弹出三选一（共享父 Prompter）              |
| worktree 子代理                 | `WebAllowlist` / `WebFetchPrompter` 从父复制 |
| 「始终允许」                    | 写入**项目** config（非 worktree 路径）      |

**自动化**：`go test ./internal/agent/spawn/...`（若有新增用例）绿。

## 6. web_fetch 工具验收（FR-6）

### AC-6.1 结构

| 检查                  | 预期                         |
| --------------------- | ---------------------------- |
| `WebFetchTool.Perm`   | setup 注册时注入             |
| `fetchURL` 签名       | 无 `allowlist []string` 参数 |
| `web_fetch_policy.go` | 不存在                       |

### AC-6.2 行为不变项

| 检查                       | 预期                  |
| -------------------------- | --------------------- |
| `web.fetch_enabled: false` | `ErrDisabled`         |
| LRU cache                  | 命中缓存不发起 HTTP   |
| 跨域重定向                 | 返回 `REDIRECT:` 提示 |
| `normalizeURL`             | 行为不变              |

### AC-6.3 自动化

`go test ./internal/tool/builtin/web_fetch/...` 绿。

## 7. permission 单测矩阵（FR-7.1）

| 用例组       | 覆盖                                       |
| ------------ | ------------------------------------------ |
| SSRF         | loopback、私有 IP、metadata、DNS 失败      |
| allowlist    | 精确、通配、大小写、端口剥离               |
| readonly/ask | 命中无 prompt；未命中 mock prompter 三选一 |
| auto         | 空 allowlist + 公网通过                    |
| 非交互       | `ErrNeedTTY`                               |

`go test ./internal/permission/... -run Web` 绿。

## 8. TUI 单测（FR-7.4）

| 检查                      | 预期                                |
| ------------------------- | ----------------------------------- |
| `HandleWebFetchPromptKey` | `1`/`2`/`3` 与 `a`/`s`/`d` 映射正确 |
| overlay 状态              | `WebFetchPromptRequest` 关闭后恢复  |

`go test ./internal/ui/tui/... -run WebFetch` 绿。

## 9. OpenTelemetry 日志关联验收（FR-8）

> 设计：[DESIGN.md §13](DESIGN.md#13-opentelemetry-日志关联fr-8)

### AC-8.1 默认关闭（NFR-5）

| 检查                                      | 预期                                 |
| ----------------------------------------- | ------------------------------------ |
| 默认（无 YAML `tracing`、未传 `--trace`） | `trace.Enabled()` 为 `false`         |
| `trace.Start`                             | 返回原 `ctx`；`SpanFromContext` 无效 |
| agent 热路径日志                          | **无** `trace_id` / `span_id` 字段   |

### AC-8.2 配置入口（CLI + YAML）

| 检查                                                 | 预期                                                           |
| ---------------------------------------------------- | -------------------------------------------------------------- |
| YAML 键                                              | `tracing.enabled`、`tracing.exporter`、`tracing.otlp_endpoint` |
| CLI                                                  | `--trace`、`--trace-exporter`、`--trace-otlp-endpoint`         |
| 合并                                                 | 用户级 → 项目级 YAML；与现有 config 一致                       |
| 优先级                                               | 显式 CLI flag（`Changed`）**覆盖** YAML                        |
| YAML `enabled: true`                                 | 无 `--trace` 时也启用 span                                     |
| `--trace=false`（显式）                              | 覆盖 YAML `enabled: true`，关闭 trace                          |
| [`configs/example.yaml`](../../configs/example.yaml) | 含 `tracing` 段及注释                                          |
| 桌面端预留                                           | 直接设置 `config.Config.Tracing` 生效（无需 cobra）            |

**自动化**：`go test ./internal/config/... -run 'Tracing|Trace|CLIDerived'` 绿。

### AC-8.3 全局 `logging.L()` 自动注入（FR-8.3）

> 更新时机细则：[DESIGN.md §13.9.1](DESIGN.md#1391-trace_id--span_id-更新时机)

| 检查                             | 预期                                                             |
| -------------------------------- | ---------------------------------------------------------------- |
| 机制                             | `traceCore` 包装 + `logctx` goroutine 栈                         |
| active span 内任意 `logging.L()` | 自动含 `trace_id`、`span_id`                                     |
| 无 span（启动、idle）            | 日志**无** trace 字段；行为等同 v0.1.4                           |
| **不要求**改调用点               | agent、permission、mcp、context、llm、TUI 等包保持 `logging.L()` |
| 嵌套 span                        | `span_id` 随栈顶变化；`trace_id` 同 turn 内不变                  |

**自动化**：`go test ./internal/logging/... -run 'TraceCore|Logctx'` 绿。

### AC-8.4 `logctx` 与 goroutine（FR-8.8b）

| 检查                   | 预期                                              |
| ---------------------- | ------------------------------------------------- |
| `trace.Start`          | 自动 `logctx.Push` / defer `Pop`                  |
| `runConcurrentBatch`   | 每 goroutine `defer logctx.Bind(ctx)()`           |
| spawn 异步 / TUI async | 入口 `Bind`                                       |
| 漏 Bind 的 goroutine   | 该 goroutine 内 `L()` 无 trace 字段（单测防回归） |

**自动化**：`go test ./internal/agent/... -run 'Concurrent|Logctx'` 绿。

### AC-8.5 初始化与降级（NFR-6）

| 检查                | 预期                                                  |
| ------------------- | ----------------------------------------------------- |
| `main` 启动顺序     | `logging.Setup` 后调用 `trace.Setup`；`defer` cleanup |
| `enabled: true`     | `otel.SetTracerProvider` 安装非 noop Provider         |
| exporter 初始化失败 | 降级 noop；打 `Warn`；**进程继续启动**                |
| 进程退出            | `TracerProvider.Shutdown` 被调用（无 panic）          |

**自动化**：`go test ./internal/trace/... -run Setup` 绿。

### AC-8.6 Span 埋点边界

| Span 名           | 埋点位置           | 预期                                            |
| ----------------- | ------------------ | ----------------------------------------------- |
| `run_turn`        | `runTurn` 入口     | 每次用户 turn / seeded turn 创建；`defer End`   |
| `llm.chat`        | `chatWithRecovery` | 每个 sub-round 一次；compact 重试在同一 span 内 |
| `tool.<name>`     | `executeTool`      | 工具名与 span 后缀一致（如 `tool.read`）        |
| `subagent.<type>` | `spawn.ExecuteRun` | 含 `explore` / `shell` 等类型后缀               |

| 检查            | 预期                                           |
| --------------- | ---------------------------------------------- |
| 父子 span       | 同一 turn 内共享 `trace_id`                    |
| 兄弟 span       | `llm.chat` 与 `tool.*` 的 `span_id` **不同**   |
| 子代理          | 子 `run_turn` 的 `trace_id` 与父 turn **相同** |
| 并发 tool batch | 各 goroutine 各自 `tool.*` span；无 data race  |

**自动化**：`go test ./internal/trace/... -run 'Span|Parent'` 绿。

### AC-8.7 日志串联（端到端 + 跨包）

| 检查                                                                        | 预期                                          |
| --------------------------------------------------------------------------- | --------------------------------------------- |
| YAML `tracing.enabled: true` 或 `ds-code --trace` 下完成一轮含 tool 的 turn | `ds-code.log` 中相关行含**相同** `trace_id`   |
| 同一 turn 内 LLM 与 tool 日志                                               | `trace_id` 相同；`span_id` 按操作不同         |
| `permission` deny（仅 `logging.L()`）                                       | 在 `executeTool` span 内含**相同** `trace_id` |
| `mcp` / `context` 包日志                                                    | turn 内 `logging.L()` 含 trace 字段           |
| `session_id`                                                                | 各日志行**仍显式出现**                        |

| 子代理 `run_id` | spawn 相关日志仍含 `run_id`（与 `trace_id` 并存） |

**自动化**：`go test ./internal/agent/... -run 'Trace|logsTraceID|PermissionDeny'` 绿。

### AC-8.8 业务字段与范围边界

| 检查                     | 预期                                                   |
| ------------------------ | ------------------------------------------------------ |
| DeepSeek HTTP            | 请求头**无** `traceparent`（v0.1.5 out of scope）      |
| MCP 子进程               | 无 trace 传播                                          |
| 无 span 时 `logging.L()` | 无 trace 字段（非「必须每行都有」）                    |
| `RunEphemeral`（`/btw`） | 不强制埋点；若实现 `btw` span，不得破坏 ephemeral 语义 |

### AC-8.9 Span 导出（P2，可选）

| 检查                                            | 预期                                                   |
| ----------------------------------------------- | ------------------------------------------------------ |
| `--trace-exporter=log`                          | span 以 DEBUG 写入 `ds-code.log`（需 `-vv`）；TUI 安全 |
| `--trace --trace-exporter=otlp` + 有效 endpoint | span 可到达 collector（手动或集成环境验证）            |
| `--trace-exporter=otlp` 且无 endpoint           | 启动失败，明确错误信息                                 |

> P2：无本地 collector 时，CI 可仅测配置解析与 log exporter，otlp 进手动清单 MV-4。

### AC-8.10 依赖与结构

| 检查                 | 预期                                          |
| -------------------- | --------------------------------------------- |
| 新增                 | `internal/logging/logctx.go`、`trace_core.go` |
| 新增                 | `internal/trace/`                             |
| `logging.L()` 调用点 | **无需**批量替换为 `FromContext`              |
| 全局 logger          | `traceCore` 包装；非 otelzap 全量替换         |

## 10. 手动验证清单（可选）

### MV-1 readonly + 空 allowlist

1. 项目 `web.allowlist: []`，`permission.mode: readonly`
2. 启动 TUI，让 agent 调用 `web_fetch` 访问 `https://pkg.go.dev/...`
3. 确认三选一浮层出现
4. 选「始终允许」，确认 `.ds-code/config.yaml` 含 `pkg.go.dev`
5. 再次同 URL，确认无浮层

### MV-2 auto + 空 allowlist

1. `permission.mode: auto`，`allowlist: []`
2. `web_fetch` 公网 URL 成功
3. `web_fetch` `http://127.0.0.1` 被拒绝

### MV-3 非交互

1. `ds-code -p "fetch https://example.com" --permission-mode ask`
2. 未列入主机时命令失败，错误含 TTY 提示

### MV-4 tracing 日志串联（CLI）

1. 运行 `ds-code --trace`（TUI 或 `-p`），触发一轮含 tool 的对话
2. 检查 `ds-code.log`：同一 turn 共享 `trace_id`，LLM 与 tool 的 `span_id` 不同
3. 不带 `--trace` 且 YAML 未启用时重启，新日志无 trace 字段

### MV-4b tracing 日志串联（YAML）

1. 项目 `.ds-code/config.yaml` 设置 `tracing.enabled: true`（**不传** `--trace`）
2. 启动 ds-code，触发一轮含 tool 的对话
3. 确认 `ds-code.log` 含 `trace_id`/`span_id`
4. 将 `enabled` 改回 `false`，重启后新日志无 trace 字段

### MV-5 tracing log 导出（可选）

1. `ds-code --trace --trace-exporter=log -vv`
2. 运行一轮短对话
3. 确认 stderr 出现 span JSON；`ds-code.log` 仍含 `trace_id`/`span_id`

## 11. 非目标确认

- [x] `web_search` 未改动
- [x] MCP 工具权限未改动
- [x] write/shell `Prompter` 仍为二选一
- [x] 跨域重定向未改为自动多域跟随
- [x] DeepSeek API 未注入 W3C `traceparent`
- [x] `session_id` 未被 `trace_id` 替代
- [x] OTel 默认未连接外部 collector（须 YAML 或 `--trace` 显式开启）
- [x] CLI `--trace` 可覆盖 YAML `tracing.enabled`
