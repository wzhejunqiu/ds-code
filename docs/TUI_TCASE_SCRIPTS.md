# `/tcase` 场景剧本

本文档是 **`/tcase` 内置场景剧本的权威说明**。实现代码在 [`internal/tuitest/scenarios/all.go`](../internal/tuitest/scenarios/all.go)；改剧本时请**同步更新本文档**。

Harness 总览见 [TUI_INTEGRATION_TEST.md](./TUI_INTEGRATION_TEST.md)。

## 剧本模型

一次用户提交（`/tcase run <name>` 或 Picker **Enter**）会：

1. `registry.SetActive(name)`，重置回合计数；
2. 向 TUI 提交该场景的 `Prompt`（与用户手动输入相同）；
3. Agent 每发起一次 LLM 请求，Mock Server 消费剧本中的 **一个 `Turn`**。

### `Scenario`

| 字段 | 含义 |
|------|------|
| `Name` | `/tcase run` 的参数 |
| `Prompt` | 自动填入输入框并提交的用户行 |
| `Turns` | 按顺序播放的 LLM 响应剧本 |

### `Turn`（一次 `POST /chat/completions`）

| 字段 | 含义 |
|------|------|
| `Chunks` | SSE 流式 delta（`Content` / `Reasoning`）；见 [`mockserver/sse.go`](../internal/tuitest/mockserver/sse.go) |
| `ToolCalls` | 非流式 tool_calls 回合（整段 SSE 一次下发） |
| `FinishReason` | 最后一帧 SSE 的 `finish_reason`（默认：有 tool 则为 `tool_calls`，否则 `stop`） |
| `HTTPStatus` / `ErrBody` | 非 200，模拟 API 错误 |

### `StreamChunk`

| 字段 | 含义 |
|------|------|
| `Content` | `delta.content` |
| `Reasoning` | `delta.reasoning_content` |
| `Delay` | 发送**本 chunk 之前** `Sleep`（用于肉眼观察流式） |

常量 **`streamDemoDelay = 120ms`**：演示类场景在 chunk 之间拉开间隔，便于 TUI 重绘。

### 特殊逻辑：`error-context`

剧本在 `all.go` 里只有 1 个正常 Turn，但 [`mockserver/registry.go`](../internal/tuitest/mockserver/registry.go) 在 `SetActive("error-context")` 时会设 `contextFail`：**第一次** LLM 请求先返回 HTTP 400 + `context length exceeded`，不消耗 `Turns[0]`；Agent compact 后**第二次**请求才播放 `Turns[0]`。

### Harness 工程文件

工具类场景在临时 `ProjectRoot` 下预置 [`sample.go`](../internal/tuitest/project.go)：

```go
package main

func Hello() string { return "hello" }
```

---

## 场景一览

| Name | Prompt | 回合 | 验证重点 |
|------|--------|------|----------|
| `stream-basic` | `stream basic test` | 1 | content 分片流式 → `hello world` |
| `stream-reasoning` | `reasoning stream test` | 1 | reasoning 小片流式 + 收起 + `answer` |
| `tool-read` | `read sample file` | 2 | `read_file` → 真实读盘 |
| `tool-grep` | `grep package` | 2 | `grep` |
| `tool-patch` | `apply_patch` | 2 | `apply_patch` 改 `sample.go` |
| `tool-multi` | `multi tools` | 2 | 并行 `read_file` + `grep` |
| `tool-shell` | `run shell` | 2 | `shell echo harness-ok` |
| `error-api` | `trigger api error` | 1 | `ErrLine` 展示 API 错误 |
| `error-context` | `context compact retry` | 1+注入 | compact 后成功，`after compact` |
| `md-rich` | `markdown render` | 1 | Markdown 渲染 |
| `long-stream` | `long stream` | 1 | 40×`x`，2ms/chunk |

---

## 分场景剧本

### `stream-basic`

**目的**：最小 content 流式，观察正文逐字出现。

| 步骤 | 类型 | 剧本 |
|------|------|------|
| Turn 1 | SSE | 见下表 → `finish_reason: stop` |

| # | Delay | Content |
|---|-------|---------|
| 1 | — | `hel` |
| 2 | 120ms | `lo ` |
| 3 | 120ms | `world` |

**期望 UI**：assistant 正文由 `hel` → `hello ` → `hello world` 增长。  
**Harness**：`TestHarness_streamBasic`，聊天含 `hello world`，无 `ErrLine`。

---

### `stream-reasoning`

**目的**：reasoning 小 delta 流式；thinking 时展开；content 开始后收起；正文 `answer`。

| 步骤 | 类型 | 剧本 |
|------|------|------|
| Turn 1 | SSE | reasoning 片段（每片后 120ms，首片无 delay）+ content 片段 → `stop` |

**Reasoning 片段**（顺序拼接后的全文）：

```text
Let me walk through this step by step.

First, check the ▾ label stays open.
Next, watch tiny reasoning chunks stream in.
Then content starts; thinking collapses.
```

| 阶段 | 片数 | 说明 |
|------|------|------|
| Reasoning | 28 片 | 约 3–10 字符/片，由 `reasoningParts(...)` 生成 |
| Content | 2 片 | `ans` →（120ms）→ `wer` |

**期望 UI**：

- 流式 reasoning 时：`▾ thinking …`，正文可见；
- `ans` 开始后：thinking 收为 `▸ thought for …`，reasoning 正文隐藏；
- 最终 assistant content：`answer`。

**Harness**：`TestHarness_streamReasoning`，聊天含 `answer`。

---

### `tool-read`

**目的**：两轮对话；builtin `read_file` 真实执行。

| Turn | 类型 | 剧本 |
|------|------|------|
| 1 | tool_calls | `read_file` `{"path":"sample.go"}`，`id=call_read_1` |
| 2 | SSE | content: `File read complete.` → `stop` |

**期望 UI**：工具块 running → 完成；第二轮 assistant 总结。  
**Harness**：`TestHarness_toolRead`。

---

### `tool-grep`

| Turn | 类型 | 剧本 |
|------|------|------|
| 1 | tool_calls | `grep` `{"pattern":"package","path":"."}`，`id=call_grep_1` |
| 2 | SSE | `Grep done.` → `stop` |

---

### `tool-patch`

| Turn | 类型 | 剧本 |
|------|------|------|
| 1 | tool_calls | `apply_patch`，patch 正文见下 |
| 2 | SSE | `Patch applied.` → `stop` |

**Patch 剧本**（在 `sample.go` 的 `Hello` 后增加一行 `// harness`）：

```diff
*** Begin Patch
*** Update File: sample.go
@@
 func Hello() string {
 	return "hello"
 }
+// harness
*** End Patch
```

---

### `tool-multi`

| Turn | 类型 | 剧本 |
|------|------|------|
| 1 | tool_calls（并行） | `read_file` `sample.go` + `grep` `func` @ `sample.go` |
| 2 | SSE | `Both tools finished.` → `stop` |

| Tool | ID | Arguments |
|------|-----|-----------|
| `read_file` | `call_r1` | `{"path":"sample.go"}` |
| `grep` | `call_g1` | `{"pattern":"func","path":"sample.go"}` |

---

### `tool-shell`

| Turn | 类型 | 剧本 |
|------|------|------|
| 1 | tool_calls | `shell` `{"command":"echo harness-ok"}` |
| 2 | SSE | `Shell done.` → `stop` |

---

### `error-api`

**目的**：单次请求直接 API 失败。

| Turn | HTTP | Body |
|------|------|------|
| 1 | 400 | `{"error":{"message":"bad request","type":"invalid_request"}}` |

**期望 UI**：`ErrLine` 显示错误，无正常 assistant 正文。  
**Harness**：`TestHarness_errorAPI` 要求 `ErrLine != ""`。

---

### `error-context`

**目的**：模拟 context 过长 → Agent compact → 重试成功。

| 请求次序 | 来源 | 剧本 |
|----------|------|------|
| 第 1 次 LLM | Registry 注入（不占用 `Turns`） | HTTP 400，`context length exceeded` |
| 第 2 次 LLM | `Turns[0]` | SSE content: `after compact` → `stop` |

**期望 UI**：无持久 `ErrLine`；最终聊天含 `after compact`。  
**Harness**：`TestHarness_errorContext`。

---

### `md-rich`

**目的**：单帧 Markdown  stress 渲染。

| Turn | Content（单 chunk） |
|------|---------------------|
| 1 | 见下 |

```markdown
# Title

**bold** and `code`.

```go
fmt.Println("hi")
```
```

---

### `long-stream`

**目的**：大量短 chunk，偏性能/缓冲（间隔短，肉眼几乎连成一片）。

| Turn | 剧本 |
|------|------|
| 1 | 40× content `"x"`，每片前 **2ms** delay → `stop` |

**聚合正文**：40 个 `x`。

---

## 维护约定

1. **改剧本**：编辑 `internal/tuitest/scenarios/all.go`，在 `All()` 注册，并**更新本文档**对应章节与一览表。
2. **新增场景**：补 harness 测试（`internal/tuitest/harness_test.go`）若行为可断言；在 [TUI_INTEGRATION_TEST.md](./TUI_INTEGRATION_TEST.md) 一览表加一行并链接到本文档。
3. **特殊 Mock 行为**（如 `error-context` 注入）：写在 registry，本文档「特殊逻辑」或场景内**单独注明**。
4. **流式观感**：演示场景优先用 `streamDemoDelay`；压测/长流用更短 delay（如 `long-stream` 的 2ms）。

## 相关文件

| 路径 | 作用 |
|------|------|
| `internal/tuitest/scenarios/all.go` | 剧本实现 |
| `internal/tuitest/scenarios/scenario.go` | `Scenario` / `Turn` / `StreamChunk` 类型 |
| `internal/tuitest/mockserver/sse.go` | SSE 编码 |
| `internal/tuitest/mockserver/registry.go` | 回合推进与 `error-context` 注入 |
| `internal/tuitest/harness_test.go` | 自动化断言 |
