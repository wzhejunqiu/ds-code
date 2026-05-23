# task

## 概述

启动一个**只读子 Agent**，在独立会话中并行探索代码库，最终将摘要文本返回给主 Agent。用于大范围检索、多路径调查等不宜塞入主上下文的任务。

## 注册与可见性

| 模式 | 注册条件 |
|------|----------|
| agent | `setup.Deps.LLM != nil` |
| plan | **不注册** |

## 参数 Schema

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `prompt` | string | 是 | 给子 Agent 的详细任务说明 |
| `description` | string | 否 | 短标签，用于 TUI 与返回前缀 |

## 用法示例

```json
{
  "description": "Find shell tool tests",
  "prompt": "Search internal/tool/builtin for shell background job tests. List file paths and what each test covers."
}
```

## 返回格式

- 有 `description`：`[{description}]\n{summary}`
- 无 `description`：直接返回 `summary`

子 Agent 错误时 `Execute` 返回 error，主会话不收到部分摘要。

## 实现细节

源文件：[`task.go`](task.go)

### 并发控制

`taskSem`：带缓冲 channel，容量 `tools.task.max_parallel`（默认 3）。`Acquire`/`Release` 包裹整次 `Run`。

### 会话与存储

1. 从 `context` 读取 `agent.ToolInvocation`（`SessionID`、`ToolCallID`）。
2. `Subagent.CreateRun` 写入 `subagent_runs`（关联 parent session / tool call）。
3. `subagent.Run`：
   - 新建 `permission.Engine("readonly")`
   - `RegisterExploreTools`（read_file、grep、glob、list_dir）
   - `agent.Runner.RunTurn`，最多 `min(8, agent.max_turns)` 轮
4. `FinishRun` 更新状态；`OnSubagentStart` / `OnSubagentEnd` 回调刷新 TUI。

主会话 **不** 将子 Agent 消息注入 `BuildAPIContext`，仅保留一条 `task` 工具结果。

### 摘要截断

`subagent.trimSummary` 使用 `tools.task.summary_max_chars`（默认 16000）。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `tools.task.max_parallel` | 3 | 同时运行的子 Agent 数 |
| `tools.task.summary_max_chars` | 16000 | 返回主会话的摘要最大字符数 |
| `agent.max_turns` | 25 | 子 Agent 轮次上限为 `min(8, max_turns)` |

## 权限与安全

- **PermissionLevel**：`Low`（工具本身不直接写盘）
- 子 Agent 权限模式 **readonly**，无 shell / apply_patch / write_file
- 仍需 LLM token 成本；并行度过高会放大费用

## 设计思想

- **并行探索、串行决策**：子 Agent 做广度搜索，主 Agent 综合后行动。
- **上下文隔离**：避免子对话污染主 transcript，只回传摘要。
- **可恢复 UI**：`parent_tool_call_id` 关联 TUI `/resume` 与子代理详情面板。

## 相关代码

- [`task.go`](task.go)
- [`task_test.go`](task_test.go)
- [`explore.go`](../../register/explore.go) — 子代理工具集（`RegisterExploreTools`）
- [`subagent/runner.go`](../../../agent/subagent/runner.go)、[`subagent/README.md`](../../../agent/subagent/README.md)
- [`subagentstore/`](../../../session/subagentstore/)
