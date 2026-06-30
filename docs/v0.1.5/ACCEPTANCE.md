# v0.1.5 验收标准

> 版本：v0.1.5  
> 状态：规划中  
> 更新日期：2026-06-30  
> 需求：[REQUIREMENTS.md](REQUIREMENTS.md) · 设计：[DESIGN.md](DESIGN.md)

## 1. 总体验收

- [ ] 版本号标记为 v0.1.5（发布打 `v0.1.5` tag 时由 ldflags 注入）
- [ ] `make test` / `make lint` / `make vet` 通过
- [ ] `CHANGELOG.md` 含 allowlist 语义 **breaking** 说明
- [ ] `configs/example.yaml` `web.allowlist` 注释已更新
- [ ] `internal/tool/builtin/web_fetch/web_fetch_policy.go` 已删除

**自动化冒烟**：

```bash
go test -race -count=1 ./internal/permission/... ./internal/config/... ./internal/tool/builtin/web_fetch/... ./internal/ui/tui/...
make test
```

## 2. allowlist 语义验收（FR-0）

### AC-2.1 readonly/ask + allowlist 命中

| 检查 | 预期 |
|------|------|
| `web.allowlist: [example.com]` | 访问 `https://example.com/...` **无** prompt |
| 通配 `*.github.io` | `foo.github.io` 命中 |
| SSRF 主机（localhost、169.254.x.x） | **拒绝**，即使在 allowlist |

### AC-2.2 auto 模式

| 检查 | 预期 |
|------|------|
| 空 allowlist + 公网 URL | SSRF 通过即放行 |
| 私有 IP / loopback | 拒绝 |

### AC-2.3 空 allowlist 行为变更（breaking）

| 检查 | v0.1.4 | v0.1.5 |
|------|--------|--------|
| readonly + `allowlist: []` + 公网 | 拒绝 | 三选一 prompt |
| auto + `allowlist: []` + 公网 | 拒绝 | 放行（SSRF 通过） |

## 3. 三选一 Prompter 验收（FR-2、FR-4）

### AC-3.1 TUI 交互

| 检查 | 预期 |
|------|------|
| 未列入主机触发 web_fetch | 浮层显示 host / URL |
| 按 `1` 或 `a` | 本次请求完成；同 host 重定向不二次 prompt |
| 按 `2` 或 `s` | 本次完成；`.ds-code/config.yaml` 含新 host |
| 按 `3` 或 `d` | `ErrRejected`；工具返回拒绝 |
| write/shell prompt | 仍为 y/n 二选一（未回归） |

### AC-3.2 非 TUI 降级

| 检查 | 预期 |
|------|------|
| `ds-code -p` + 未列入主机 | `ErrNeedTTY` |
| stdin 模式（ask + 有 TTY 无 TUI） | 打印三选项；读 stdin 生效 |

### AC-3.3 AllowAlways 内存生效

| 检查 | 预期 |
|------|------|
| 选「始终允许」后同会话再次访问 | **无** prompt |
| 重启进程后 | config 已合并，**无** prompt |

## 4. config 持久化验收（FR-3）

### AC-4.1 文件写入

| 检查 | 预期 |
|------|------|
| 目标路径 | `<project>/.ds-code/config.yaml` |
| 权限 | `0600` |
| 去重 | 重复「始终允许」不重复条目 |
| 保留 | 已有 allowlist 项不丢失 |
| 用户级 config | `~/.ds-code/config/config.yaml` **不变** |

### AC-4.2 自动化

`go test ./internal/config/... -run WebAllowlist` 绿。

## 5. Engine 与子代理验收（FR-1、FR-5）

### AC-5.1 `Engine.Check(web_fetch)`

| 检查 | 预期 |
|------|------|
| `Check("web_fetch", {"url": "https://..."})` | 走 `CheckWebFetch` |
| deny 日志 | `classifyDeny` 可区分 web_fetch |

### AC-5.2 启动注入

| 检查 | 预期 |
|------|------|
| `runner.Perm.WebAllowlist` | 等于合并后 `cfg.Web.Allowlist` |
| TUI | `WebFetchPrompter` 已挂载 |

### AC-5.3 子代理

| 检查 | 预期 |
|------|------|
| bubble/inherit 子代理 web_fetch | 可弹出三选一（共享父 Prompter） |
| worktree 子代理 | `WebAllowlist` / `WebFetchPrompter` 从父复制 |
| 「始终允许」 | 写入**项目** config（非 worktree 路径） |

**自动化**：`go test ./internal/agent/spawn/...`（若有新增用例）绿。

## 6. web_fetch 工具验收（FR-6）

### AC-6.1 结构

| 检查 | 预期 |
|------|------|
| `WebFetchTool.Perm` | setup 注册时注入 |
| `fetchURL` 签名 | 无 `allowlist []string` 参数 |
| `web_fetch_policy.go` | 不存在 |

### AC-6.2 行为不变项

| 检查 | 预期 |
|------|------|
| `web.fetch_enabled: false` | `ErrDisabled` |
| LRU cache | 命中缓存不发起 HTTP |
| 跨域重定向 | 返回 `REDIRECT:` 提示 |
| `normalizeURL` | 行为不变 |

### AC-6.3 自动化

`go test ./internal/tool/builtin/web_fetch/...` 绿。

## 7. permission 单测矩阵（FR-7.1）

| 用例组 | 覆盖 |
|--------|------|
| SSRF | loopback、私有 IP、metadata、DNS 失败 |
| allowlist | 精确、通配、大小写、端口剥离 |
| readonly/ask | 命中无 prompt；未命中 mock prompter 三选一 |
| auto | 空 allowlist + 公网通过 |
| 非交互 | `ErrNeedTTY` |

`go test ./internal/permission/... -run Web` 绿。

## 8. TUI 单测（FR-7.4）

| 检查 | 预期 |
|------|------|
| `HandleWebFetchPromptKey` | `1`/`2`/`3` 与 `a`/`s`/`d` 映射正确 |
| overlay 状态 | `WebFetchPromptRequest` 关闭后恢复 |

`go test ./internal/ui/tui/... -run WebFetch` 绿。

## 9. 手动验证清单（可选）

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

## 10. 非目标确认

- [ ] `web_search` 未改动
- [ ] MCP 工具权限未改动
- [ ] write/shell `Prompter` 仍为二选一
- [ ] 跨域重定向未改为自动多域跟随
