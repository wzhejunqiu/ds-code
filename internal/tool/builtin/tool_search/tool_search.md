# tool_search

## 概述

按工具名称返回**完整 JSON Schema** 与描述，供 LLM 在调用「延迟加载」的内建写工具前先获取参数定义。本身为只读元数据查询，不执行目标工具。

## 注册与可见性

| 模式 | 注册 |
|------|------|
| agent | `RegisterAgentExtras`（始终注册） |
| plan | **不注册** |

仅当 Registry 中存在至少一个 `DeferredTool`（`tools.defer_builtin` 配置了写工具名）时，LLM 才需要本工具；但注册不依赖是否配置了 defer。

## 参数 Schema

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tool_name` | string | 是 | 已注册工具的名称，如 `shell`、`apply_patch` |

## 用法示例

```json
{"tool_name": "apply_patch"}
```

典型流程：

1. LLM 在工具列表中看到 `apply_patch` 的 stub schema（仅 `_note` 字段）。
2. 调用 `tool_search` 获取完整 `patch` 参数说明。
3. 再调用 `apply_patch` 传入正确参数。

## 返回格式

```text
Tool: {name}
Description: {description}
Schema:
{indented JSON schema}
```

未知工具名：`unknown tool: {name}`。

## 实现细节

源文件：[`tool_search.go`](tool_search.go)

1. `Registry.Get(tool_name)` 校验工具存在。
2. `Registry.FullSchema(tool_name)` 取完整 schema（绕过 defer stub）。
3. 实现 `ReadOnlyTool` 与 `ConcurrencySafeTool`，可与其它探索工具并行。

## 配置项

| 键 | 说明 |
|----|------|
| `tools.defer_builtin` | 列出需延迟暴露 schema 的内建工具名；与本工具配合使用 |

## 权限与安全

- **PermissionLevel**：`Low`
- 只读；不泄露工作区文件内容，仅暴露已注册工具的 schema 文本

## 设计思想

- **控制 prompt token**：写工具 schema 往往较大；defer + `tool_search` 按需加载。
- **Execute 不变**：defer 只影响发给 LLM 的 `Definitions()`，不改变工具运行时行为。
- **与 MCP defer 分离**：`tools.defer_mcp` 由 MCP 层处理；本工具仅服务内建 Registry。

## 相关代码

- [`tool_search.go`](tool_search.go)
- [`tool_search_test.go`](tool_search_test.go)
- [`deferred_wrapper.go`](../../deferred_wrapper.go) — stub schema
- [`registry.go`](../../registry.go) — `FullSchema`、`HasDeferredTools`
- [`setup/setup.go`](../../setup/setup.go) — 注册顺序
