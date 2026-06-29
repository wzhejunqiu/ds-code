# DeepSeek 模型调用规范（ds-code）

> 文档版本：v2.0  
> 更新日期：2026-05-16  
> 状态：与 [PLAN.md](PLAN.md) v0.12+、[DESIGN.md](DESIGN.md) 配套，实现 `internal/llm/deepseek` 时以本文为准

## 官方参考

- [首次调用 API](https://api-docs.deepseek.com/zh-cn/)
- [思考模式](https://api-docs.deepseek.com/zh-cn/guides/thinking_mode)
- [Tool Calls](https://api-docs.deepseek.com/zh-cn/guides/tool_calls)
- [接入 Agent 工具](https://api-docs.deepseek.com/zh-cn/)（Claude Code、OpenCode 等）
- [**对话补全 API**](https://api-docs.deepseek.com/zh-cn/api/create-chat-completion)（请求/响应、`usage`、`max_tokens`）

---

## 端点与认证

| 参数 | 值 |
|------|-----|
| `base_url`（OpenAI 兼容） | `https://api.deepseek.com` |
| `base_url`（Beta，strict 模式） | `https://api.deepseek.com/beta` |
| API Key | **仅环境变量**：`DS_CODE_DEEPSEEK_API_KEY` → `DEEPSEEK_API_KEY`；缺失则报错（[CONFIG.md §3.1](CONFIG.md#31-deepseek-api-keyllmapi_key)） |
| 对话接口 | `POST /chat/completions` |

**Go 客户端**：[`github.com/sashabaranov/go-openai`](https://github.com/sashabaranov/go-openai) + `WithBaseURL("https://api.deepseek.com")`

**密钥安全**：禁止写入仓库；日志/错误栈脱敏；子进程 env 最小化。

---

## 模型选型（V4）

| model ID | 上下文长度 | 单次最大输出 | ds-code 建议用途 |
|----------|------------|--------------|------------------|
| `deepseek-v4-pro` | **1,048,576**（1Mi，2²⁰） | **393,216**（384Ki） | **默认**（Agent 编程） |
| `deepseek-v4-flash` | **1,048,576**（1Mi） | **393,216**（384Ki） | 快速试探、低成本 |
| `deepseek-chat` | 弃用 | 弃用 | 迁移 → `deepseek-v4-flash` |
| `deepseek-reasoner` | 弃用 | 弃用 | 迁移 → `deepseek-v4-flash` + thinking |

> 上下文/输出上限为产品设计常量（两档 V4 相同），实现时以 [对话补全 API](https://api-docs.deepseek.com/zh-cn/api/create-chat-completion) 与[模型 & 价格](https://api-docs.deepseek.com/zh-cn/) 为准；若官方变更，更新 `internal/llm/deepseek/limits.go`。

```go
// internal/llm/deepseek/limits.go（建议）
const (
    ContextWindowTokens   = 1_048_576  // 1Mi（2^20），非 10^6
    MaxOutputTokens       = 393_216    // 384Ki（384×1024）；对应 API max_tokens 上限
)
```

**请求预算**（简化，模型无关）：

- **compact 触发**：条件 **A/B/C**（见下方）；计费展示 `SessionBilledTokens = prompt + completion` **不单独**作为 compact 依据。
- **`max_tokens`**：`min(配置, MaxOutputTokens)`；默认 **16384**，硬顶 **393,216**。若 API 返回上下文过长，触发 compact 并重试。

---

## 上下文预算（compact 与计费）

### 计费累计（状态栏）

每次 `chat/completions` 响应后，将 `usage` 累加到当前 `session`：

```go
func SessionBilledTokens(s Session) int {
    return s.PromptTokensTotal + s.CompletionTokensTotal
}
```

### compact 触发（A / B / C）

阈值 `T = compact_threshold_ratio × ContextWindowTokens`（默认 **838,861**）。在 `PrepareRequest` 评估，**同一 `PrepareRequest` 最多 compact 一次**（详见 [PLAN.md](PLAN.md#会话-token计费累计-vs-compact-触发)）：

| 条件 | 规则 |
|------|------|
| **A** | 用户轮首个子轮次：`CountBreakdown(BuildAPIContext).Total() >= T`；缓存至用户轮结束 |
| **B** | `prompt_tokens_total >= T`（**不含** completion） |
| **C** | API 返回上下文过长 → compact → 重试一次 |

| 项 | 说明 |
|----|------|
| compact 后 | 计费累计 **不清零**；压缩 API 上下文层 |
| 手动 | `/compact` 同 `CompactAPIContext`，仅改 API 层 |
| Phase 1–2 | `ShouldCompact` 恒 false |

### `internal/context` 职责

**入口**：`PrepareRequest(session) (*APIContextView, maxTokens, error)`

1. 若 `ShouldCompact(...)` → `CompactAPIContext(session)`（至多一次）
2. `view := BuildAPIContext(session)`
3. `maxTokens := min(cfg.MaxTokens, MaxOutputTokens)`
4. 返回 view，供 client 发送

### 可选：本地 Tokenizer（非预算路径）

> [internal/tokenizer/deepseek/tokenizer.go](../../internal/tokenizer/deepseek/tokenizer.go)

| 用途 | 说明 |
|------|------|
| 调试 | `cmd/count-tokens` |
| 工具截断 | 配置 `context.truncate_by: tokenizer` 时可选；**默认按字符/字节上限** |
| **compact 条件 A** | 用户轮首个子轮次 `CountBreakdown`（缓存复用） |
| **`/context` 六分项** | 用户执行 `/context` 时展示（可与 A 同源） |

依赖 CGO + `scripts/fetch-tokenizers-lib.sh`；无 CGO 时条件 A 与 `/context` 用 `CharCounter` 并标注「估算」。

#### 输入：`APIContextView`

由 `BuildAPIContext(session)` 构造，表示**下一次** `POST /chat/completions` 将使用的输入（与 [PLAN.md · 双层消息模型](PLAN.md#双层消息模型持久化-vs-api-上下文) 一致，非 SQLite 全量历史）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `SystemPrompt` | `string` | 固定 system 基座（拼装前） |
| `AgentsMD` | `string` | `AGENTS.md` |
| `Rules` | `string` | rules 片段 |
| `Skills` | `string` | 当前激活 skill |
| `GitSnapshot` | `string` | 当前分支、默认分支、Git user、`git status`、最近 5 条提交（可选；计入 **System prompt** 分项） |
| `ToolsJSON` | `string` | 请求体 `tools` 数组 JSON（与 API 序列化一致） |
| `Messages` | `[]Message` | **仅** `user` / `assistant` / `tool`；**不含** `role=system`（system 由上行字段合并为一条） |

**子代理（Canonical）**：`task` 工具摘要 → `Messages` 中 `role=tool`（工具名 `task`）。

> **拼装原则**：`mergeSystem` 合并为单条 `role=system`；`Messages` 仅含 user/assistant/tool；子代理 `task` 结果在 `Messages` 中为 `role=tool`。

#### 请求前流程（`PrepareRequest`）

**每次**主对话 LLM 请求前（含 tool 子轮次）：

1. 若 `ShouldCompact`（A/B）→ `CompactAPIContext`（至多一次；usage 累加、失败降级同 [PLAN.md](PLAN.md)）
2. `view := BuildAPIContext(session)`
3. `maxTokens := min(cfg.MaxTokens, MaxOutputTokens)`
4. 发送 `chat/completions`；响应后累加 `usage`

#### 工具结果截断

- 默认：`TruncateToolResult(body, maxChars int)` 按**字符**上限（`context.tool_result_max_chars`）。
- 可选：`truncate_by: tokenizer` 时用 `tokenizer/deepseek.Count` 精确截断。

#### `CountBreakdown(view *APIContextView) (ContextBreakdown, error)`

**用途**：**compact 条件 A**（用户轮内缓存）与 **`/context` 展示**；不在每个 tool 子轮次重复计算。

**含义**：将「下一次请求」的 API 输入拆成 **6 个互斥分项**（本地估算，可能与实际上屏 `usage` 略有偏差）。

| 分项（UI 名） | 计数对象 | 说明 |
|---------------|----------|------|
| `SystemPrompt` | `mergeSystem(...)` 合并后的单条 system | 含 Git |
| `Tools` | `ToolsJSON` | |
| `Rules` | `Rules` 字段 | **展示行**；已并入 System 时 UI 标注「已含于 System」，**不参与 Total 求和** |
| `Skills` | `Skills` 字段 | 未激活为 0；同上 |
| `Subagents` | `Messages` 中 `tool` 且来自 `task` | |
| `Conversation` | `Messages` 其余部分 | 不含 task tool、不含 system |

```go
func CountBreakdown(view *APIContextView) (ContextBreakdown, error) {
    tok, err := deepseek.Default() // 首期；未来可注入 Counter 接口
    // mergeSystem + Count(ToolsJSON) + Count(serializeMessages) 拆分 Subagents/Conversation
}

type ContextBreakdown struct {
    SystemPrompt int
    Tools        int
    Rules        int // 展示用
    Skills       int
    Subagents    int
    Conversation int
    Window       int // 1_048_576
}
func (b ContextBreakdown) Total() int {
    return b.SystemPrompt + b.Tools + b.Subagents + b.Conversation
}
func (b ContextBreakdown) PercentOfWindow(part int) float64 { ... }
func (b ContextBreakdown) PercentOfTotal(part int) float64 { ... }
```

- **不变式**：`Total() == Count(mergeSystem) + Count(ToolsJSON) + Count(非 task 的 Messages) + Count(task tool Messages)`（单元测试）。
- **与 session 累计的关系**：六分项描述**下一次 prompt 构成**；会话累计描述**历史已消耗**；二者分开展示，勿混为 compact 依据。

---

## ds-code 默认配置

YAML 键、CLI、环境变量及合并优先级见 **[CONFIG.md](CONFIG.md)**；全量示例见 [`configs/example.yaml`](../../configs/example.yaml)。

与 API 直接相关的默认：`llm.model` = **`deepseek-v4-pro`**，`llm.thinking.type` = **`enabled`**，`llm.reasoning_effort` = **`max`**；Subagent（`llm.subagent`）默认 **`deepseek-v4-flash`** / thinking **`disabled`** / reasoning **`high`**。`llm.max_tokens` = **16384**（硬顶 393216），`context.window_tokens` = **1048576**，`context.compact_threshold_ratio` = **0.80**。UI 费用估算单位为 **人民币（¥）**，按 `messages` / `subagent_messages` 持久化的价格快照计算（见 [CONFIG.md §5.1](CONFIG.md#51-llm--deepseek-客户端)）。

**允许切换的模型**（`/mode` 与配置校验白名单）：`deepseek-v4-pro`、`deepseek-v4-flash`（弃用名自动迁移见上表）。

---

## 运行时：模型与思考强度

> 参考：[思考模式](https://api-docs.deepseek.com/zh-cn/guides/thinking_mode)

### 模型（默认 `deepseek-v4-pro`）

| 方式 | 说明 |
|------|------|
| 配置文件 | `llm.model`（[CONFIG.md §5.1](CONFIG.md#51-llm--deepseek-客户端)） |
| TUI **`/mode`** | `/mode` 显示当前；`/mode deepseek-v4-flash` 切换；建议 **per-session** 写入 `sessions.model` |
| 启动参数 | `ds-code --model deepseek-v4-flash`（[CONFIG.md §4](CONFIG.md#4-cli-参数)） |

### 思考模式开关 `thinking.type`

| 值 | 说明 |
|----|------|
| `enabled` | **默认**；返回 `reasoning_content` |
| `disabled` | 关闭思维链；无 `reasoning_content` |

配置项 `llm.thinking.type`（[CONFIG.md](CONFIG.md)）；TUI 可选 `/thinking on|off`。

### 思考强度 `reasoning_effort`（默认 `max`）

| 值 | 说明 |
|----|------|
| `max` | **默认**；Agent 编程场景；官方对 Agent 类请求亦倾向 `max` |
| `high` | 普通对话、低成本试探 |
| `low` / `medium` | API 映射为 `high`（兼容） |
| `xhigh` | API 映射为 `max`（兼容） |

| 方式 | 说明 |
|------|------|
| 配置文件 | `llm.reasoning_effort`（默认 `max`，见 [CONFIG.md](CONFIG.md)） |
| TUI **`/effort`** | `/effort max` \| `/effort high`（**per-session** → `sessions.reasoning_effort`） |

**请求组装**（每次 `POST /chat/completions`，字段以[官方 API](https://api-docs.deepseek.com/zh-cn/api/create-chat-completion)为准）：

```json
{
  "model": "deepseek-v4-pro",
  "messages": [],
  "tools": [],
  "tool_choice": "auto",
  "stream": true,
  "stream_options": { "include_usage": true },
  "max_tokens": 16384,
  "thinking": { "type": "enabled" },
  "reasoning_effort": "max",
  "user_id": "<optional: datadir.Identifier()，见下表>"
}
```

| 参数 | ds-code 约定 |
|------|----------------|
| `model` | session 或配置；仅 `deepseek-v4-pro` \| `deepseek-v4-flash` |
| `max_tokens` | `min(配置, 393_216)`；默认 **16384**，硬顶 **393,216**；上下文过长由 API 报错 → compact 重试 |
| `stream` | **true**（TUI 必需） |
| `stream_options.include_usage` | **true**；末 chunk 在 `data: [DONE]` 前带完整 `usage` |
| `thinking` | session / 配置；Go SDK 可放 `extra_body` |
| `reasoning_effort` | `high` \| `max`，默认 `max` |
| `temperature` / `top_p` | 思考模式 **enabled** 时不传或忽略（官方不生效） |
| `frequency_penalty` / `presence_penalty` | **deprecated**，不传 |
| `user_id` | 可选；ds-code 使用 **`datadir.Identifier()`**：`hex(sha256(UUIDv4 + whoami + "ds-code"))`，持久化于 `~/.ds-code/identifier`；主会话、`/btw`、compact、web_fetch 等全部共用，利于跨会话 prompt cache 命中（[API 说明](https://api-docs.deepseek.com/zh-cn/api/create-chat-completion)） |

**`finish_reason: length`**：达到 `max_tokens` 或上下文上限；Runner 应提示用户缩小任务或 `/compact`。

---

## 思考模式（Thinking Mode）

DeepSeek V4 **默认开启**思考模式。思维链通过 `reasoning_content` 返回，与 `content` 同级。

| 参数（OpenAI 格式） | 说明 |
|---------------------|------|
| `thinking.type` | `enabled` / `disabled`；默认 `enabled` |
| `reasoning_effort` | `high` / `max`；**默认 `max`**；`low`/`medium` → `high`，`xhigh` → `max` |

### 实现约束（必须遵守）

1. **思考模式下** `temperature`、`top_p`、`presence_penalty`、`frequency_penalty` **不生效**（传了不报错）。`thinking=enabled` 时配置/UI 应隐藏或禁用这些项。
2. **无工具调用的轮次**：下一轮可不带历史 `reasoning_content`（API 会忽略）。会话仍可存储，回传时按「该 Turn 是否发生过 tool_calls」决定。
3. **有工具调用的轮次**（ds-code 主路径）：同一 Turn 内每次回注 API 的 `assistant` 消息 **必须完整携带** `content`、`reasoning_content`、`tool_calls`；否则 API 返回 **400**。

```go
// 回注 assistant 消息（有 tool_calls 时 reasoning_content 不可丢）
messages = append(messages, assistantMsg) // Role, Content, ReasoningContent, ToolCalls
```

4. **流式**：分别累积 `delta.reasoning_content` 与 `delta.content`；TUI 可折叠展示思维链。
5. **默认强度**：未显式设置时使用配置/ session 的 `reasoning_effort`（默认 **`max`**）。

### 请求示例（思考 + Tool Calls + 流式）

```json
{
  "model": "deepseek-v4-pro",
  "messages": [],
  "tools": [],
  "stream": true,
  "reasoning_effort": "max",
  "thinking": { "type": "enabled" }
}
```

Go SDK：`thinking` 放请求体顶层或 `extra_body`（与 [官方 Python 样例](https://api-docs.deepseek.com/zh-cn/guides/thinking_mode) 一致）。

---

## Tool Calls

- **思考 / 非思考模式**均支持；ds-code Agent 循环走 **思考模式 + tools**。
- 执行流程：user → assistant（`tool_calls`）→ tool（`tool_call_id` + result）→ assistant（最终 `content`）→ 循环。

### strict 模式（Beta）

- 配置 `llm.strict_tools: true` 时，**全部** `chat/completions`（含 compact 摘要、btw 若将来启用 tools）使用 `base_url: https://api.deepseek.com/beta`
- 每个 tool：`function.strict: true`
- JSON Schema 须符合 [strict 规范](https://api-docs.deepseek.com/zh-cn/guides/tool_calls)
- **Phase 2 启用**（与 `apply_patch` 同期），降低 tool 参数 JSON 格式错误

---

## `internal/llm/deepseek/client.go` 职责

| 职责 | 说明 |
|------|------|
| 模型名规范化 | 弃用名 → V4 映射 + 日志告警 |
| 流式解析 | 累积 `reasoning_content`、`content`、`tool_calls` |
| 消息序列化 | 持久化/回传保留 `reasoning_content`（tool call 路径） |
| 错误重试 | 429/5xx 指数退避 |
| 用量统计 | 解析 `usage`（含 `completion_tokens_details.reasoning_tokens`） |
| compact 触发 | `ShouldCompact` A/B → `CompactAPIContext`（至多一次/Prepare）；C = API 过长重试 |

---

## Token 用量（usage）

每次 `chat/completions` 响应（含流式最后一帧）携带 `usage` 对象。ds-code **状态栏展示三项累计**：**输入**、**输出**、**缓存命中**。

> 参考：[上下文硬盘缓存](https://api-docs.deepseek.com/zh-cn/guides/kv_cache)（默认开启，无需额外参数）

| API 字段 | 状态栏含义 | 说明 |
|----------|------------|------|
| `prompt_tokens` | **输入** | 当次请求输入 token 总量 |
| `completion_tokens` | **输出** | 当次模型生成 token（含思维链，以 API 计费为准） |
| `prompt_cache_hit_tokens` | **缓存命中** | 从磁盘缓存命中的输入 token（低价计费） |
| `prompt_cache_miss_tokens` | （可选） | 未命中缓存；**`prompt_tokens = hit + miss`**（[API](https://api-docs.deepseek.com/zh-cn/api/create-chat-completion)） |
| `completion_tokens_details.reasoning_tokens` | （可选） | 思维链 token；费用/分析用，状态栏可不单独展示 |
| `total_tokens` | （可选） | `prompt + completion`；内部校验 |

**TUI 展示映射**（session 累计值）：

```
输入 {prompt_tokens_total} · 输出 {completion_tokens_total} · 缓存 {prompt_cache_hit_tokens_total}
```

**累计规则**：

- 每次 LLM 调用（含 tool call 子轮次）将当次 `usage` 对应字段 **累加**到当前 `session_id`
- `prompt_cache_hit_tokens` 缺失时按 `0` 处理（兼容旧响应或非缓存场景）
- 流式：在含 `usage` 的末 chunk 解析后更新

**持久化**（`sessions` 表建议字段）：

| 列名 | 说明 |
|------|------|
| `model` | 当前模型，默认 `deepseek-v4-pro` |
| `reasoning_effort` | 思考强度，默认 `max` |
| `thinking_type` | `enabled` / `disabled` |
| `compact_summary` | API 层摘要文本（**非**替换 messages 行） |
| `compact_up_to_message_id` | 已被摘要覆盖的历史水位线 |
| `prompt_tokens_total` | 累计 `prompt_tokens` |
| `completion_tokens_total` | 累计 `completion_tokens` |
| `prompt_cache_hit_tokens_total` | 累计 `prompt_cache_hit_tokens` |

`messages` 表可选存单次的 `prompt_tokens` / `completion_tokens` / `prompt_cache_hit_tokens` 便于审计。

**TUI**：详见 [PLAN.md · TUI](PLAN.md#tui--cli-体验)。

---

## 上下文预算与工具侧限制

在 **1,048,576 上下文 / 393,216 最大输出** 下，ds-code 通过 **工具层 + 注入层** 控制膨胀（详见 [PLAN.md · 上下文预算](PLAN.md#上下文预算与工具设计)）：

| 机制 | 默认策略 | 目的 |
|------|----------|------|
| `read_file` | `start`/`end` 闭区间 + `max_lines`（默认 2000）+ `max_bytes`（默认 2MiB） | 默认读整个文件（最多 max_lines 行），按行扫描 |
| `grep` | `head_limit` 默认 200 条匹配 | 限制搜索结果体积 |
| `@` 引用 | 单文件 + 总计 ≤ `at_reference_max_chars` | 预加载不撑爆 prompt |
| `apply_patch` | 拒绝单 patch 变更行数超阈值（可配置） | 防止超大 diff 回注 |
| tool 结果 | 超长截断 + 提示「已截断，可调整 start/end 续读」 | 边界内可审计 |
| 子代理 | 摘要 ≤ 4K tokens 量级 | 主会话省窗口 |
| 自动 compact | `PrepareRequest`：条件 A 或 B（阈值默认 **838,861**） | 摘要写入 session；**messages 历史不变**；API 层 = 摘要 + 近 N **用户轮** |
| `/compact` | 用户手动 | 同自动 compact，仅 API 层 |
| `max_tokens` | 默认 16Ki/轮；长回答可配置上调，硬顶 **393,216** | 为 prompt 留足空间 |

---

## 会话存储字段

`messages` 表建议包含：`id`, `session_id`, `role`, `content`, `reasoning_content`, `tool_calls_json`, `tool_call_id`, `prompt_tokens`, `completion_tokens`, `prompt_cache_hit_tokens`, `created_at`。

`/btw`：**不**写入 `messages`；`user_id` 与主会话相同（`datadir.Identifier()`，见 [PLAN.md · /btw](PLAN.md#btw-快速提问不进入主对话)）。

---

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-05-16 | 初版：V4 模型、思考模式、`reasoning_content` 回传、strict Beta |
| 2026-05-16 | 补充 `usage` 累计规则与 session 持久化字段（供 TUI 状态栏） |
| 2026-05-16 | 状态栏三项：输入 / 输出 / 缓存命中（`prompt_cache_hit_tokens`） |
| 2026-05-16 | `/mode` 切换模型（默认 v4-pro）；`/effort` 与配置项 `reasoning_effort`（默认 max） |
| 2026-05-16 | V4 上下文 1Mi（1,048,576）、最大输出 384Ki；预算校验与工具截断 |
| 2026-05-16 | 上下文长度更正为 **1,048,576**（非 1,000,000） |
| 2026-05-16 | 预计算绑定 `internal/tokenizer/deepseek`（HF tokenizer，禁止启发式） |
| 2026-05-16 | 自动 compact：EnsureBudget 每轮 LLM 前 ≥80%；双层模型，历史 messages 不受影响 |
| 2026-05-16 | `/clear`：新 session_id，不删历史；见 PLAN.md |
| 2026-05-16 | `/btw` 旁路请求：不写 messages，默认无 tools |
| 2026-05-16 | `CountBreakdown` 六分项：system/tools/rules/skills/subagents/conversation |
| 2026-05-16 | 重写 `budget`：`APIContextView`、`CountPrompt`/`CountBreakdown` 算法与互斥不变式 |
| 2026-05-16 | **v1.7**：`GitSnapshot`、merge 单条 system、`task`→Subagents、compact usage/降级、`strict_tools`、btw `cache_scope` |
| 2026-05-16 | **v1.8**：取消发送前 `CountPrompt`；compact 由 **session API usage 累计** ≥80% 触发 |
| 2026-05-16 | **v1.9**：`/context` 恢复 **六分项** `CountBreakdown`（按需、仅展示）；与会话累计分层 |
| 2026-05-16 | **v2.0**：compact **A/B/C**；`SessionBilledTokens`；条件 A 使用 `CountBreakdown`（用户轮缓存） |
