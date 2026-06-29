# web_fetch

## 概述

在配置允许时，通过 HTTP(S) GET 获取 URL 正文，将 HTML 转为 Markdown，并使用轻量模型（默认 `deepseek-v4-flash`）按 `prompt` 分析页面内容。

## 注册与可见性

| 模式 | 注册条件 |
|------|----------|
| plan / agent | `web.fetch_enabled: true`（默认）且目标主机在 `web.allowlist` 中 |

## 参数 Schema

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `url` | string | 是 | `http://` 或 `https://` URL（默认 80 端口 http 自动升级 https） |
| `prompt` | string | 是 | 要从页面提取或分析的信息 |

## 返回格式

- **成功**：模型分析结果（纯文本）
- **跨域重定向**：`REDIRECT: <url>` + 说明文字；须用该 URL 重新请求
- **错误**：`Err*` 文案

## 实现细节

源文件：[`web_fetch.go`](web_fetch.go)、[`fetch.go`](fetch.go)、[`analyze.go`](analyze.go)、[`cache.go`](cache.go)、[`convert.go`](convert.go)

### 流水线

1. `normalizeURL`（http 默认端口 → https、去 fragment、host 小写）
2. 内存 LRU 缓存查找（key=url，gzip 压缩正文，15min TTL，总容量 50MiB）
3. 未命中则 HTTP GET（手动处理重定向；跨域返回 REDIRECT；同域跟随最多 10 次）
4. **无重定向**的响应写入 LRU；发生过重定向的不缓存
5. HTML → Markdown（`html-to-markdown`）
6. LLM 分析（`web.fetch_model`）

### 多层校验

与 [`web_fetch_policy.go`](web_fetch_policy.go) 相同：allowlist、私有 IP 阻断、DNS rebinding 防护。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `web.fetch_enabled` | true | 是否注册并允许调用（默认开；`allowlist` 为空时仍拒绝所有主机） |
| `web.allowlist` | [] | 允许的主机列表；**空列表表示全部拒绝** |
| `web.fetch_model` | `deepseek-v4-flash` | 页面分析模型 |
| `web.fetch_cache_ttl` | `15m` | LRU 单条 TTL |
| `web.fetch_cache_max_bytes` | `52428800` | LRU 总容量（压缩后） |

## 权限与安全

- **PermissionLevel**：`Medium`
- 防 SSRF：allowlist + 私有 IP 阻断
- 认证/私有页面无法抓取；跨域重定向不自动跟随

## 相关代码

- [`web_fetch.go`](web_fetch.go)、[`usage.prompt`](usage.prompt)
- [`display.go`](../../display.go) — `FormatWebFetchDisplay`
