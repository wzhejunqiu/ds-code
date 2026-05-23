# web_fetch

## 概述

在配置允许时，通过 HTTP(S) GET 获取 URL 正文，供 Agent 阅读文档、API 说明等外部资源。

## 注册与可见性

| 模式 | 注册条件 |
|------|----------|
| plan / agent | `web.fetch_enabled: true` |

## 参数 Schema

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `url` | string | 是 | `http://` 或 `https://` URL |

## 用法示例

```json
{"url": "https://example.com/docs/page"}
```

需在 `web.allowlist` 中允许主机，例如 `example.com` 或 `*.example.com`。

## 返回格式

```text
HTTP {status_code}
{body}
```

响应体读取上限 **512 KiB**；整体可能再经 `context.TruncateToolResult` 截断。

## 实现细节

源文件：[`web_fetch.go`](../web_fetch.go)、[`web_fetch_policy.go`](../web_fetch_policy.go)

### 多层校验

| 阶段 | 检查 |
|------|------|
| 请求前 | `validateFetchURLHost`：主机须在 allowlist；禁止 localhost、metadata 等 |
| 重定向 | 最多 10 次；每次检查新 host 仍在 allowlist |
| 拨号时 | `validateResolvedFetchHost`：DNS 解析后拒绝私有 IP、链路本地、169.254.169.254 |

`isBlockedFetchHost` 对主机名做 DNS 查找，私有 IP 直接拒绝。

### HTTP 客户端

- 超时 30s
- 自定义 `Transport.DialContext` 在连接建立前再次校验解析后的 IP

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `web.fetch_enabled` | false | 是否注册并允许调用 |
| `web.allowlist` | [] | 允许的主机列表；支持 `*.domain.com`；**空列表表示全部拒绝** |

## 权限与安全

- **PermissionLevel**：`Medium`
- 防 SSRF：allowlist + 私有 IP 阻断 + localhost 阻断
- Plan 模式可用，但仍受 `permission.mode` 约束（与 agent 相同 ask 策略）

## 设计思想

- **默认关闭**：外网访问需显式开启并配置白名单。
- **双重主机检查**：URL 解析阶段与 TCP 拨号阶段各验一次，缓解 DNS rebinding。
- **只读 GET**：不支持 POST/自定义头，降低数据外泄面。

## 相关代码

- [`web_fetch.go`](../web_fetch.go)
- [`web_fetch_policy.go`](../web_fetch_policy.go)
- [`web_fetch_test.go`](../web_fetch_test.go)、[`web_fetch_policy_test.go`](../web_fetch_policy_test.go)
- [`display.go`](../../display.go) — `FormatWebFetchDisplay`
