# spawn

子代理（sub-agent）生命周期编排：路由、工具池过滤、执行、后台管理与结果回传。

[`agent` 内置工具](../../tool/builtin/agent/agent.go) 的所有业务逻辑委托本包；TUI slash skill（`context: fork`）也走同一套 Fork 路径。主 Runner 循环见 [`../README.md`](../README.md)。

## 设计理念

### 单一入口

所有 spawn 路径收敛到 [`Service.Handle`](./service.go)（及 [`FromSkill`](./skill.go) 对 Fork 的薄封装）。路由、持久化、执行、通知不在工具层重复实现。

### 复用主 Runner，而非另写循环

子代理不实现独立的 LLM 循环，而是构造**受限版** [`agent.Runner`](../runner.go)（[`ExecuteRun`](./execute.go)）：

- 同一套 [`chatWithRecovery`](../recovery.go)、[`tool_orchestration`](../tool_orchestration.go)、Hook
- 独立的 [`sessionStore`](./execute.go) 适配器 → [`subagentstore`](../../session/subagentstore/types.go)（与主 session 隔离）
- 独立的 [`context.Service`](../../context/service.go)（AgentOverlay / ForkView）

这样主 agent 与子 agent 行为一致，bug 修复与 recovery 策略只需维护一处。

### 三种执行形态

| 形态 | SpawnKind | 何时选用 | 父工具返回 |
|------|-----------|----------|------------|
| **Sync** | `SpawnSync` | 默认；短任务同步等待 | 摘要 inline 或 spill 指针 |
| **Async** | `SpawnAsync` | `run_in_background` / verification 强制 / Sync 超时 promote | `{"status":"async_launched",...}` |
| **Fork** | `SpawnFork` | 交互模式 + 省略 `subagent_type` + `fork_enabled` | 同 Sync |

Fork 的核心目标是**共享父会话 prompt cache 前缀**：子 agent 复制父 API 消息 + 固定占位 tool_result，仅最后 user directive 不同。

### 安全边界

- **禁止嵌套 spawn**：所有子 agent 工具池 Layer 1 过滤 `agent`
- **Fork 不可递归**：Fork 子消息含 `fork-boilerplate` 标签；`QuerySourceFork` 下再次 spawn 被拒绝
- **权限 bubble**：Fork 子 agent 权限询问冒泡到父 TUI Prompter
- **Worktree 路径校验**：删除前 `ValidatePath`，防止越界
- **已读集合隔离**：子 agent 独立 `session_id` / `Runner`，`apply_patch` read-guard 不从父 session 继承；Fork 可从 seed 消息水合历史 read

### 父会话不阻塞

Async 与 Sync→Async promote 让长任务脱离父 tool_call 等待。完成结果通过 **NotificationQueue** 三级优先级回灌主会话，而非阻塞 LLM 子轮次。

## 包结构

| 文件 | 说明 |
|------|------|
| [service.go](./service.go) | `Service.Handle` / `runSync` / `finishSync` / `finishAsync` |
| [router.go](./router.go) | `Route`：Fork / Sync / Async 决策 |
| [execute.go](./execute.go) | `ExecuteRun`、`sessionStore` 适配器 |
| [registry.go](./registry.go) | 内置 agent 类型定义 |
| [fork.go](./fork.go) | `BuildForkMessages`、`ForkBoilerplate` |
| [toolpool.go](./toolpool.go) | `FilterToolRegistry` 三层过滤 |
| [background.go](./background.go) | `BackgroundManager` 后台 goroutine |
| [notify.go](./notify.go) | `NotificationQueue`、XML 格式 |
| [output.go](./output.go) | `DeliverResult` inline vs spill |
| [prompts.go](./prompts.go) | Explore/Plan/verification system overlay |
| [memory.go](./memory.go) | `~/.ds-code/agent-memory` 注入 |
| [skill.go](./skill.go) | `FromSkill`（Skill context:fork） |
| [model.go](./model.go) | `ResolveModel`、`resolveSubagentMaxTurns` |
| [context.go](./context.go) | `QuerySource`、`DetachSpawnContext` |
| [worktree_cleanup.go](./worktree_cleanup.go) | worktree 创建失败 / 会话 / TTL 清理 |

## 架构

```
AgentTool.Execute  →  ../../tool/builtin/agent/agent.go
       │
       ▼
Service.Handle     →  service.go
       │
       ▼
Route              →  router.go
       │
       ├─ Background ──► background.go ──► execute.go (ExecuteRun)
       └─ Sync ──► service.go (runSync) ──► execute.go
                          │
                          ▼
                   output.go / notify.go
```

### Service 依赖

定义见 [`service.go`](./service.go)。

```go
type Service struct {
    Registry, Perm, ParentReg, LLM, Store, Cfg
    BackgroundManager *BackgroundManager
    NotifyQueue       *NotificationQueue
    Hooks             *agent.HookManager      // app 组装时注入
    ParentContext     *ctxpkg.Service           // Fork 读父 API 视图
    Worktrees         *worktree.Manager
}
```

[`NewService`](./service.go) 在 [`AgentTool`](../../tool/builtin/agent/agent.go) 构造时创建；[`app.newRunner`](../../cmd/ds-code/app/runner.go) 随后注入 `Hooks`、`ParentContext`，并接线 [`Runner.DrainNotifications*`](../runner_turn.go)。

## 路由（[`router.go`](./router.go)）

[`Route`](./router.go)(ctx, params, inv, reg, cfg, interactive) 决策树：

```
1. Registry.Resolve(subagent_type)   // 空 → general-purpose

2. subagent_type 省略 && fork_enabled && interactive?
   ├─ QuerySource 已是 Fork → 错误
   ├─ 父消息已是 Fork 子 agent → 错误（递归）
   ├─ run_in_background → 错误
   └─ SpawnFork

3. def.ForceBackground || params.run_in_background?
   ├─ 是 → SpawnAsync
   └─ 否 → SpawnSync
```

### Params（LLM 工具参数）

定义见 [`router.go`](./router.go)。

```go
type Params struct {
    Description     string // 短标签，UI 与 tool 返回用
    Prompt          string // 子 agent 任务正文
    SubagentType    string // general-purpose | Explore | Plan | verification
    Model           string // 可选覆盖
    RunInBackground bool
    Isolation       string // "worktree"（仅 general-purpose）
}
```

## 内置 Agent 类型（[`registry.go`](./registry.go)）

| Type | ReadOnly | ForceBackground | 工具池 | System |
|------|----------|-----------------|--------|--------|
| `general-purpose` | 否 | 否 | `*` − agent | 父 overlay + memory |
| `Explore` | 是 | 否 | 禁 write/patch/agent | exploreOverlay |
| `Plan` | 是 | 否 | 禁 write/patch/agent | planOverlay |
| `verification` | 是 | **是** | 禁 write/patch/agent | verificationOverlay + VERDICT |
| `fork`（合成） | 否 | 否 | 继承父池 − agent | 父 rendered system + memory |

[`ListTypes`](./registry.go)() 供 [`AgentTool.Schema`](../../tool/builtin/agent/agent.go) 生成 enum，**不含**合成类型 `fork`。

## Handle 主流程（[`service.go`](./service.go)）

```
Route → ResolveModel
     → [worktree] Worktrees.Create + ValidatePath
     → Store.CreateRun（记录 parent_session、tool_call_id、spawn_kind）
     → HookSubagentStart + OnSubagentStart
     → [Fork] 校验 ForkContext
     → [Background] BackgroundManager.Start → 立即返回 async_launched
     → [Sync] runSync
```

### runSync 与 Sync→Async promote

实现：[`service.go`](./service.go)（`runSync`、`waitPromoted`）。

`auto_background_after > 0`（默认 120s）时：

1. goroutine 内跑 `ExecuteRun`
2. `select`：完成 → `finishSync`；超时 → `SetRunBackground` + `RegisterPromoted` + 返回 `async_launched`
3. 超时后 `waitPromoted` 在后台等完成 → `finishAsync` 入队通知

`auto_background_after <= 0` 时纯阻塞 Sync，无 promote。

### finishSync vs finishAsync

| | [`finishSync`](./service.go) | [`finishAsync`](./service.go) |
|---|------------|-------------|
| Store | `FinishRun(completed/error)` | 同上（含 killed） |
| 结果 | [`DeliverResult`](./output.go) → tool 返回字符串 | [`NotificationQueue.Enqueue`](./notify.go) |
| Worktree | 失败时 immediate cleanup | 失败/killed 时 cleanup |
| UI | `OnSubagentEnd` 同步触发 | promote 路径异步触发 |

## ExecuteRun（[`execute.go`](./execute.go)）

子 agent 的执行核心：组装受限 Runner 并跑一轮。

### 1. 权限引擎

| permMode / 类型 | 行为 |
|-----------------|------|
| ReadOnly 类型 | [`permission.NewEngine`](../../permission/engine.go)(`"readonly"`, workspace) |
| `inherit` / `bubble` | 继承父 Perm；worktree 时 rebind workspace，保留 Prompter |
| worktree | workspace = worktree 路径 |

### 2. 工具池（[`toolpool.go`](./toolpool.go)）

```
FilterToolRegistry  →  toolpool.go
  Layer 1: 跳过 agent（全局）
  Layer 2: 跳过 def.DisallowedTools
  Layer 2.5: background 时仅 asyncAllowed 白名单
```

后台白名单：`read_file`, `glob`, `grep`, `list_dir`, `diagnostics`, `web_fetch`, `web_search`, `shell`, `write_file`, `apply_patch`。

Worktree 子 agent：[`tool.RebindRegistryPerm`](../../tool/registry.go)(childReg, perm) 绑定新 workspace。

### 3. sessionStore 适配器

[`sessionStore`](./execute.go) 实现 [`session.Store`](../../session/store.go) 接口，底层映射到 [`subagentstore.Store`](../../session/subagentstore/types.go) 的单条 `Run`：

- `Get` / `ListMessages` / `AppendMessage` / `AddUsage` 均按 `runID` 读写
- 不支持 `Create` / `ListSessions`（子 agent 无多 session 概念）

### 4. context.Service 分支

**常量子 agent**（Explore / Plan / general-purpose / verification）：

```go
ctxSvc.AgentOverlay = SystemPromptOverlay(def) + FormatAgentMemory(def.Type)  // prompts.go, memory.go
ctxSvc.VerificationMode = (def.Type == "verification")
childRunner.RunTurn(ctx, sess.ID, run.Prompt, cb)  // ../runner_turn.go
```

**Fork**：

```go
forkMsgs := BuildForkMessages(...)  // fork.go
seedForkMessages → store           // execute.go
ctxSvc.ForkView = BuildForkAPIContext(...)  // ../../context/service.go
childRunner.RunTurnSeeded(ctx, sess.ID, cb)
```

Fork 的 [`ForkContext`](../fork_context.go) / [`RenderedSystem`](../fork_context.go) 由父 [`enrichAgentForkContext`](../runner.go) 或 [`FromSkill`](./skill.go) 注入 ctx。

### 5. 子 Runner 参数

```go
childRunner := &agent.Runner{
    LLM, Tools: childReg, Perm, Sessions: store, Context: ctxSvc,
    Cfg, MaxTurns: resolveSubagentMaxTurns(cfg),  // 默认 8
    Out: io.Discard, Hooks, ForSubagent: true,
}
```

摘要返回：[`trimSummary`](./execute.go)(FinalContent, cfg)，超长加 [`SubagentSummaryTruncated`](../text.go) 后缀。

## Fork 消息构造（[`fork.go`](./fork.go)）

为 prompt cache 前缀字节一致：

```go
const ForkPlaceholder = "Fork started — processing in background"  // 固定，不可 per-child 变化
```

[`BuildForkMessages`](./fork.go) 步骤：

1. 复制父 API 消息至含 `tool_calls` 的 assistant（含**全部** sibling tool_use，不仅是 fork 那一个）
2. 对每个 `parentToolCalls` 追加 `role=tool`，content = `ForkPlaceholder`
3. 追加 `role=user`：`ForkBoilerplate` + `[directive: "<json-encoded prompt>"]`

[`IsInForkChild`](./fork.go) 检测 user 消息是否含 `fork-boilerplate`，用于递归防护。

Fork 子 agent **继承父 ThinkingType**；常量子 agent 默认 `disabled`。

## Skill Fork（[`skill.go`](./skill.go)）

`.ds-code/skills/<name>/SKILL.md` front matter 含 `context: fork` 时，TUI 调用 [`Service.FromSkill`](./skill.go)：

- 不解析 LLM `agent` 工具参数；prompt = skill body
- 需 `fork_enabled` + interactive
- 手动注入 [`ForkContext`](../fork_context.go)（从 [`ParentContext.BuildAPIContext`](../../context/service.go)）
- [`QuerySource`](./context.go) = `skill:fork`
- 走 [`runSync`](./service.go)（无 background）

## 结果交付（[`output.go`](./output.go)）

[`ShouldSpillResult`](./output.go) 逻辑：

Spill 阈值：`len > 1 MiB` 或 utf8 runes > `summary_max_chars`（默认 16000）。

| 模式 | 父工具 / 通知内容 |
|------|------------------|
| Inline | `[description]\n{summary}` 或通知 `<result>` |
| Spill | `{"status":"completed","output_file":"..."}` + `SavedResultHint`（[`SavedResultHint`](../../toolresult/project_data_hint.go)） |

Spill 文件含 `status:`、`error:`（如有）、summary 正文。异步通知 spill 时在 `</task-notification>` 后追加同格式 hint。

## 通知（[`notify.go`](./notify.go)）

### 优先级

| 优先级 | 注入时机 | 写入方式 |
|--------|----------|----------|
| `PrioNow` | 父 idle，`DrainNotifications` | 拼接到下条 user **文本前** |
| `PrioNext` | 同上，RunTurn 开头 | 同上 |
| `PrioLater` | 父回合工具 batch 后 | `role=user` 追加 `<task-notification>` 到 session |

[`notificationPriority`](./notify.go)(ctx)：[agent.InActiveTurn](../context.go)(ctx) → `PrioLater`，否则 `PrioNow`。

### XML 格式

```xml
<task-notification>
  <task-id>...</task-id>
  <tool-use-id>...</tool-use-id>
  <status>completed|failed|killed</status>
  <summary>...</summary>
  <result>...</result>          <!-- inline 时 -->
  <output-file>...</output-file> <!-- spill 时 -->
  <worktree>...</worktree>      <!-- 可选 -->
</task-notification>
```

[`NotificationQueue`](./notify.go) 按 `AgentID` 去重，防止重复通知。

## 后台管理（[`background.go`](./background.go)）

- [`Start`](./background.go)：[DetachSpawnContext](./context.go)(parent) 启动 goroutine，父 ctx 取消不 kill 已启动的后台 agent
- [`Kill`](./background.go)(runID)：取消 context
- [`RegisterPromoted`](./background.go) / [`CompletePromoted`](./background.go)：Sync→Async promote 路径追踪 in-flight 任务
- [`RunningCount`](./background.go) / [`List`](./background.go)：TUI 状态展示

## Worktree 隔离

仅 `general-purpose` + `isolation: worktree`：

- 路径：`{project_data}/worktrees/wt-{slug}/`
- 创建：sparse checkout（默认 `/*`）+ symlink `node_modules`/`.venv`/`vendor`
- 分支名：`ds-code/agent-{slug}`

清理（[`worktree_cleanup.go`](./worktree_cleanup.go)）：

| 函数 | 时机 |
|------|------|
| [`cleanupWorktreeImmediate`](./worktree_cleanup.go) | run 失败 / killed / CreateRun 回滚 |
| [`CleanupSessionWorktrees`](./worktree_cleanup.go) | 主 session 结束 |
| [`CleanupExpiredWorktrees`](./worktree_cleanup.go) | app 启动 + TTL 过期（默认 24h） |

Worktree 创建逻辑见 [`service.go`](./service.go)（[`worktreeSlug`](./service.go)、[`worktreeOpts`](./service.go)）。

## Agent Memory（[`memory.go`](./memory.go)）

持久化路径：`~/.ds-code/agent-memory/{agentType}/`

Slot 文件：`user.md`, `feedback.md`, `project.md`, `reference.md`

- [`LoadAgentMemory`](./memory.go)：取最近修改的 3 个 slot，总长 ≤ 7000 字符
- [`FormatAgentMemory`](./memory.go)：包在 `<agent-memory agent_type="...">` 注入子 agent system
- [`SaveAgentMemory`](./memory.go)：追加写入 slot（单文件上限 32 KiB）

## System Overlay（[`prompts.go`](./prompts.go)）

| 类型 | 注入内容要点 |
|------|-------------|
| Explore | 只读探索，报告路径与行号 |
| Plan | 只读架构分析，输出 Critical Files + 分步计划 |
| verification | 对抗性验证，末行 `VERDICT: PASS \| FAIL \| PARTIAL` |

## Model 与 MaxTurns（[`model.go`](./model.go)）

**Model 优先级**：[`ResolveModel`](./model.go) — 工具参数 → 类型定义（非 `inherit`）→ `llm.subagent.model` → `llm.model`

**MaxTurns**：[`resolveSubagentMaxTurns`](./model.go)（默认 8），与主 Runner 的 `agent.max_turns`（25）独立。

## Context 辅助（[`context.go`](./context.go)）

```go
type QuerySource string  // agent:builtin:* | skill:fork — 诊断与递归防护

DetachSpawnContext(parent)  // WithoutCancel + 新 CancelFunc
```

## 与父 Runner 的集成

[`cmd/ds-code/app/runner.go`](../../cmd/ds-code/app/runner.go)：

```go
runner.DrainNotifications = func(ctx) string {
    return formatNotifications(svc, PrioNow, PrioNext)  // 拼 XML 到 user 文本前
}
runner.DrainNotificationsLater = func(ctx, sessionID) {
    for _, n := range svc.DrainNotifications(PrioLater) {
        store.AppendMessage(user, n.FormatXML())
    }
}
svc.Hooks = runner.Hooks
svc.ParentContext = ctxSvc
```

[`AgentTool`](../../tool/builtin/agent/agent.go) 信号量：`tools.agent.max_parallel`（默认 3）限制并发 [`Handle`](./service.go) 调用。

父 ctx 通过 [`WithTurnCallbacks`](../context.go) / [`WithToolInvocation`](../context.go) / [`WithForkContext`](../fork_context.go) 传入 spawn；子 agent 工具 UI 经 [`SubagentToolCallbacks`](../context.go) 转发。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `tools.agent.fork_enabled` | true | 省略 type 时走 Fork |
| `tools.agent.auto_background_after` | 120 | Sync 超时 promote（秒）；0=禁用 |
| `tools.agent.max_parallel` | 3 | 并发 agent 工具数 |
| `tools.agent.summary_max_chars` | 16000 | inline 摘要上限 |
| `tools.agent.worktree_ttl` | 24h | 过期 worktree 清理 |
| `tools.agent.worktree_sparse_paths` | `["/*"]` | sparse checkout |
| `tools.agent.worktree_symlink_dirs` | node_modules 等 | symlink 加速 |
| `llm.subagent.max_turns` | 8 | 子 agent 子轮次上限 |
| `llm.subagent.model` | — | 子 agent 默认模型 |
| `llm.subagent.fallback_model` | — | 子 agent 5xx 降级 |

## 测试

| 文件 | 覆盖 |
|------|------|
| [spawn_test.go](./spawn_test.go) | Route 决策、Registry |
| [fork_test.go](./fork_test.go) | BuildForkMessages、递归检测 |
| [execute_test.go](./execute_test.go) | ExecuteRun、sessionStore |
| [execute_worktree_test.go](./execute_worktree_test.go) | worktree 路径 rebind |
| [background_test.go](./background_test.go) | Async、promote、Kill |
| [notify_test.go](./notify_test.go) | 优先级、XML、去重 |
| [output_test.go](./output_test.go) | inline/spill 阈值 |
| [skill_test.go](./skill_test.go) | FromSkill 前置条件 |
| [memory_test.go](./memory_test.go) | slot 读写与截断 |
| [integration_test.go](./integration_test.go) | Handle 端到端 |
| [persist_test.go](./persist_test.go) | subagentstore 消息持久化 |
| [context_test.go](./context_test.go) | QuerySource、DetachSpawnContext |
| [thinking_test.go](./thinking_test.go) | Fork ThinkingType 继承 |

运行：`go test -race -count=1 ./internal/agent/spawn/...`

## 扩展指南

**新增 agent 类型**：在 [`registry.go`](./registry.go) 的 `registerBuiltins` 中 `Register`，指定 `Tools`/`DisallowedTools`/`ReadOnly`/`ForceBackground`；如需行为约束，在 [`prompts.go`](./prompts.go) 增加 overlay。

**新增 spawn 入口**：应调用 [`ExecuteRun`](./execute.go) 或复用 [`Route`](./router.go) + [`Handle`](./service.go) 后半段，避免复制 Runner 构造逻辑。

**禁止**：子 agent 工具池放行 `agent`；Fork 子 agent 再次 Fork。
