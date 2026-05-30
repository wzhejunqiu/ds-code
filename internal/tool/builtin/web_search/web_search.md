# web_search

## 概述

**占位工具**：Schema 与权限级别已定义，但尚未接入搜索提供商；当前**不会**通过 `setup` 注册到 Registry。

## 注册与可见性

| 状态 | 说明 |
|------|------|
| 代码存在 | `web_search.go` |
| 运行时 | `RegisterAgentExtras` 未注册；`_ = d.Cfg.Web.SearchEnabled` 占位，待 provider 接入后再接线 |

即使手动注册，在 `web.search_enabled: false` 或未配置 provider 时，`Execute` 也会报错。

## 参数 Schema（预留）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `query` | string | 是 | 搜索查询字符串 |

## 预期用法（未来）

```json
{"query": "Go 1.22 slices package changes"}
```

## 当前行为

1. `web.search_enabled == false` → `web_search is disabled (set web.search_enabled: true)`
2. 已启用但无 provider → `web_search provider not configured; use web_fetch with a known URL`

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `web.search_enabled` | false | 未来总开关 |

## 权限与安全

- **PermissionLevel**：`Medium`（设计中与 `web_fetch` 同级）
- 启用后仍需考虑 API 密钥、查询日志、结果截断与 allowlist 策略

## 设计思想

- **接口先行**：保持工具名与 Schema 稳定，便于后续接入 Brave/Google 等 provider。
- **明确降级路径**：未实现时引导使用 `web_fetch` + 已知 URL，避免静默空结果。

## 相关代码

- [`web_search.go`](web_search.go)
- [`setup/setup.go`](../../setup/setup.go) — 注册占位说明
