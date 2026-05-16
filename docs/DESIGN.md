# ds-code 详细设计文档

> 文档版本：v1.0  
> 更新日期：2026-05-16  
> 状态：实现基线  
> 上游文档：[PLAN.md](PLAN.md)（路线图与验收）、[llm-deepseek.md](llm-deepseek.md)（模型/API 契约）

---

## 1. 文档目的与读者

本文在 [PLAN.md](PLAN.md) 之上展开**可实现的模块边界、数据结构、核心流程与接口**，供 Phase 0–7 编码与 Code Review 使用。模型字段、usage 累计、思考模式等**以 [llm-deepseek.md](llm-deepseek.md) 为权威**；本文引用但不重复全部 API 表格。

| 读者 | 关注章节 |
|------|----------|
| 核心开发 | §3–§8、§10 |
| TUI/交互 | §9 |
| 安全/审计 | §11 |
| 运维/发布 | §12、§13 |

---

## 2. 产品目标与约束

### 2.1 目标

构建 Go 原生 CLI **ds-code**：终端 AI 编程代理，首期单 Provider **DeepSeek V4**，对标 Claude Code / Codex 的 Agent 范式。

### 2.2 硬约束

| 约束 | 说明 |
|------|------|
| 上下文窗口 | **1,048,576** tokens（1Mi，2²⁰） |
| 单次最大输出 | **393,216** tokens（384Ki） |
| 历史不可删 | `messages` 表只增；compact/clear **不删除**历史行 |
| compact 触发 | **会话 API usage 累计** ≥ 80% 窗口；**不在**每次发请求前本地 Count |
| `/context` 六分项 | 用户执行 `/context` 时**按需** `CountBreakdown`；与 compact **解耦** |
| 单二进制 | `go build` 产出 `ds-code`；MCP/LSP 子进程外置 |

### 2.3 非目标（首期）

IDE 插件、云端 session 同步、Hooks 插件、git worktree 隔离、多 Provider。见 PLAN §非目标。

### 2.4 仓库现状

| 已有 | 待建 |
|------|------|
| `go.mod`、`internal/tokenizer/deepseek`、`cmd/count-tokens`、embed 词表 | `cmd/ds-code`、`internal/agent`、`session`、`ui` 等主体 |

---

## 3. 系统架构

### 3.1 分层图

```mermaid
flowchart TB
  subgraph presentation [Presentation]
    CLI[cmd/ds-code]
    TUI[internal/ui]
    Slash[internal/ui/slash]
  end

  subgraph application [Application]
    Runner[internal/agent.Runner]
    SubRunner[internal/agent/subagent]
    Ephemeral[RunEphemeral /btw]
  end

  subgraph domain [Domain]
    CtxPkg[internal/context]
    Perm[internal/permission]
    Sess[internal/session]
    CP[internal/checkpoint]
    Tools[internal/tool]
    MCP[internal/mcp]
  end

  subgraph infrastructure [Infrastructure]
    LLM[internal/llm/deepseek]
    Tok[internal/tokenizer/deepseek]
    Config[internal/config]
    DB[(SQLite)]
  end

  CLI --> TUI
  TUI --> Slash
  TUI --> Runner
  Runner --> CtxPkg
  Runner --> Perm
  Runner --> Sess
  Runner --> CP
  Runner --> Tools
  Runner --> LLM
  Runner --> SubRunner
  Ephemeral --> LLM
  Tools --> MCP
  Perm --> Tools
  Sess --> DB
  CtxPkg --> Sess
  CtxPkg --> Tok
  LLM --> Config
```

### 3.2 依赖规则

1. **`internal/agent`** 可依赖 context、session、tool、permission、llm；**不**依赖 ui。
2. **`internal/ui`** 依赖 agent、session、context（只读 Build/CountBreakdown）。
3. **`internal/llm`** 不依赖 agent/ui；通过接口 `llm.Client` 注入。
4. **循环依赖禁止**：context ↔ session 通过接口 `session.Reader` / `session.Writer` 解耦。

### 3.3 核心类型（包级）

```go
// internal/agent/runner.go
type Runner struct {
    LLM         llm.Client
    Tools       *tool.Registry
    Perm        *permission.Engine
    Sessions    session.Store
    Checkpoints checkpoint.Store
    Context     context.Service   // BuildAPIContext, PrepareRequest, Compact
    Mode        RunMode           // Agent | Plan
    MaxTurns    int
    Config      *config.Config
}

type RunMode int
const (
    RunModeAgent RunMode = iota
    RunModePlan
)
```

---

## 4. 领域模型

### 4.1 Session

```go
// internal/session/session.go
type Session struct {
    ID                        string
    Model                     string    // deepseek-v4-pro | flash
    ReasoningEffort           string    // high | max
    ThinkingType              string    // enabled | disabled
    CompactSummary            string
    CompactUpToMessageID      int64
    PromptTokensTotal         int64
    CompletionTokensTotal     int64
    PromptCacheHitTokensTotal int64
    CreatedAt                 time.Time
    UpdatedAt                 time.Time
}

func SessionUsedTokens(s Session) int {
    return int(s.PromptTokensTotal + s.CompletionTokensTotal)
}
```

### 4.2 Message（历史层）

```go
type Message struct {
    ID                   int64
    SessionID            string
    Role                 string // user | assistant | tool | system(事件)
    Content              string
    ReasoningContent     string
    ToolCallsJSON        string
    ToolCallID           string
    PromptTokens         int64  // 可选，当次
    CompletionTokens     int64
    PromptCacheHitTokens int64
    CreatedAt            time.Time
}
```

- **只增**：`Store.AppendMessage`；无 `Update`/`Delete`（审计与 resume）。
- **system 事件**：如 checkpoint rewind，`role=system`，不进 API 的 mergeSystem。

### 4.3 APIContextView（API 层快照）

```go
// internal/context/view.go
type APIContextView struct {
    SystemPrompt string
    AgentsMD     string
    Rules        string
    Skills       string
    GitSnapshot  string
    ToolsJSON    string
    Messages     []llm.Message
}
```

与 [llm-deepseek.md · APIContextView](llm-deepseek.md#输入apicontextview) 一致。

### 4.4 ContextBreakdown（展示层）

```go
type ContextBreakdown struct {
    SystemPrompt int
    Tools        int
    Rules        int // 展示行，不参与 Total
    Skills       int
    Subagents    int
    Conversation int
    Window       int // 1_048_576
}

func (b ContextBreakdown) Total() int {
    return b.SystemPrompt + b.Tools + b.Subagents + b.Conversation
}
```

---

## 5. 持久化设计

### 5.1 存储选型

- **SQLite**（`modernc.org/sqlite`），路径默认 `~/.local/share/ds-code/sessions.db`（可配置）。
- 单写连接 + WAL；`session_id` 索引。

### 5.2 表结构

#### `sessions`

| 列 | 类型 | 说明 |
|----|------|------|
| `id` | TEXT PK | UUID |
| `model` | TEXT | 默认 `deepseek-v4-pro` |
| `reasoning_effort` | TEXT | 默认 `max` |
| `thinking_type` | TEXT | `enabled` / `disabled` |
| `compact_summary` | TEXT | API 层摘要 |
| `compact_up_to_message_id` | INTEGER | 水位线 |
| `prompt_tokens_total` | INTEGER | 累计 |
| `completion_tokens_total` | INTEGER | 累计 |
| `prompt_cache_hit_tokens_total` | INTEGER | 累计 |
| `permission_mode` | TEXT | readonly/ask/auto |
| `run_mode` | TEXT | agent/plan |
| `created_at` | DATETIME | |
| `updated_at` | DATETIME | |

#### `messages`

| 列 | 类型 | 说明 |
|----|------|------|
| `id` | INTEGER PK AUTOINCREMENT | |
| `session_id` | TEXT FK | |
| `role` | TEXT | |
| `content` | TEXT | |
| `reasoning_content` | TEXT | |
| `tool_calls_json` | TEXT | |
| `tool_call_id` | TEXT | |
| `prompt_tokens` | INTEGER | 可选 |
| `completion_tokens` | INTEGER | 可选 |
| `prompt_cache_hit_tokens` | INTEGER | 可选 |
| `created_at` | DATETIME | |

索引：`(session_id, id)`、`(session_id, created_at)`。

### 5.3 双层消息模型

| 层 | 实现 | 生命周期 |
|----|------|----------|
| **历史记录层** | `messages` 全量 | 永久保留（clear 仅换 session_id） |
| **API 上下文层** | `BuildAPIContext` 内存构建 | compact 替换为「摘要 + 近 N 轮」 |

```mermaid
sequenceDiagram
  participant R as Runner
  participant C as context.Service
  participant S as session.Store
  participant L as llm.Client

  R->>S: AppendMessage(user)
  R->>C: PrepareRequest(sessionID)
  C->>S: Load session + messages
  alt SessionUsedTokens >= threshold
    C->>L: CompactAPIContext LLM call
    C->>S: Update compact_summary, watermark
    C->>S: AddUsage(compact usage)
  end
  C->>C: BuildAPIContext
  C-->>R: view, maxTokens
  R->>L: ChatCompletion(stream)
  L-->>R: deltas + usage
  R->>S: AppendMessage(assistant/tool)
  R->>S: AddUsage(usage)
```

---

## 6. 上下文服务（internal/context）

### 6.1 接口

```go
type Service interface {
    PrepareRequest(ctx context.Context, sessionID string) (view *APIContextView, maxTokens int, err error)
    BuildAPIContext(ctx context.Context, sessionID string) (*APIContextView, error)
    CompactAPIContext(ctx context.Context, sessionID string) error
    CountBreakdown(ctx context.Context, view *APIContextView) (ContextBreakdown, error)
}
```

### 6.2 PrepareRequest

```
1. sess := store.Get(sessionID)
2. if SessionUsedTokens(sess) >= ratio * ContextWindowTokens:
       CompactAPIContext(sessionID)  // 可能调 LLM，usage 累加
3. view := BuildAPIContext(sessionID)
4. maxTokens := min(cfg.LLM.MaxTokens, MaxOutputTokens)
5. return view, maxTokens
```

- **不**调用 `CountBreakdown`。
- API 返回上下文过长：Runner 捕获 → `CompactAPIContext` → **重试一次**。

### 6.3 BuildAPIContext

**输入**：`session` + `messages`（`id > compact_up_to_message_id` 或全量近端策略）+ 项目上下文加载器。

**输出**：`APIContextView`。

**拼装顺序**（固定，防 Prompt 注入）：

1. `mergeSystem(SystemPrompt, AgentsMD, Rules, Skills, GitSnapshot)` → 单条 `role=system`
2. `tools` ← `tool.Registry.SchemasForMode(runMode)` → JSON
3. `Messages`：
   - 若有 `compact_summary`：首条 assistant（元数据 `compact=true`）
   - 近端 `keep_recent_turns` 轮全文（含 `reasoning_content`、`tool_calls`）
   - `@` 引用已并入对应 user 内容

**禁止**：`Messages` 内第二条 `system`。

### 6.4 CompactAPIContext

1. 选取待摘要消息：API 层中「除 mergeSystem 等价部分 + 最近 N 轮」之外的轮次。
2. 构造 compact 专用 prompt（**脱敏**可选，S12）。
3. `llm.ChatCompletion`（非流式或流式均可；usage 必须解析）。
4. `UPDATE sessions SET compact_summary=?, compact_up_to_message_id=?`。
5. **不** UPDATE/DELETE `messages` 行。

**失败降级**：按 `created_at` 丢弃最老 API 轮次（仅影响 Build 选取范围），TUI 警告。

### 6.5 CountBreakdown（仅 /context）

- 调用时机：**仅** `slash` 处理 `context` 命令时。
- 计数器：`tokenizer.Counter` 接口；默认实现 `deepseek.Default()`。
- 无 CGO：降级 `CharCounter`（`len/4` 或 UTF-8 启发），UI 标注「估算」。

### 6.6 项目上下文加载

| 加载器 | 路径 | 缓存 |
|--------|------|------|
| `AgentsMDLoader` | 自 cwd 向上至 git 根 `AGENTS.md` | per-session 失效：cwd 变更 |
| `RulesLoader` | `.ds-code/rules/**` | glob 匹配结果缓存 |
| `SkillsLoader` | `.ds-code/skills/**/SKILL.md` + 用户目录 | 按 name 懒加载 |
| `GitSnapshot` | `git status -sb`、`git diff --stat` | 每 Turn 或 `/git` 刷新 |

---

## 7. Agent 运行时（internal/agent）

### 7.1 主循环

```mermaid
stateDiagram-v2
  [*] --> Idle
  Idle --> TurnLoop: UserSubmit
  TurnLoop --> Prepare: start sub-round
  Prepare --> LLMCall: PrepareRequest
  LLMCall --> ToolExec: tool_calls
  LLMCall --> Done: no tool_calls
  ToolExec --> Prepare: next sub-round
  ToolExec --> Done: max turns / cancel
  Done --> Idle: persist usage
```

**伪代码**：

```go
func (r *Runner) RunTurn(ctx context.Context, sessionID, userText string) error {
    userText = r.expandAtReferences(userText)
    r.Sessions.AppendMessage(sessionID, userMsg(userText))

    for turn := 0; turn < r.MaxTurns; turn++ {
        if ctx.Err() != nil { return ctx.Err() }

        view, maxTokens, err := r.Context.PrepareRequest(ctx, sessionID)
        if err != nil { return err }

        resp, err := r.LLM.Chat(ctx, llm.Request{View: view, MaxTokens: maxTokens, Stream: true})
        if isContextTooLong(err) {
            _ = r.Context.CompactAPIContext(ctx, sessionID)
            view, maxTokens, _ = r.Context.PrepareRequest(ctx, sessionID)
            resp, err = r.LLM.Chat(ctx, ...)
        }
        if err != nil { return err }

        r.Sessions.AddUsage(sessionID, resp.Usage)
        r.Sessions.AppendMessage(sessionID, assistantMsg(resp))

        if len(resp.ToolCalls) == 0 { break }

        for _, tc := range resp.ToolCalls {
            if err := r.Perm.Check(tc.Name, tc.Args); err != nil { ... }
            result, err := r.Tools.Execute(ctx, tc)
            result = context.TruncateToolResult(result, r.Config)
            r.Sessions.AppendMessage(sessionID, toolMsg(tc.ID, result))
        }
    }
    return nil
}
```

### 7.2 reasoning_content 回传规则

| 场景 | 回注 API |
|------|----------|
| 该 Turn 曾出现 `tool_calls` | assistant 必须含 `content` + `reasoning_content` + `tool_calls` |
| 纯对话无 tools | 可省略历史 `reasoning_content`（API 忽略） |

持久化层**始终存储** `reasoning_content`；`BuildAPIContext` 按规则裁剪上屏。

### 7.3 RunEphemeral（/btw）

```go
func (r *Runner) RunEphemeral(ctx context.Context, prompt string, opts EphemeralOpts) (*EphemeralResult, error)
```

| 项 | 值 |
|----|-----|
| messages | 独立切片，不写 DB |
| tools | 默认 nil |
| user_id | `btw-{uuid}` |
| usage | 默认不计入 session（`btw.count_toward_session`） |
| compact | **不**触发 |

### 7.4 Plan 模式

- `RunModePlan`：`tool.Registry` 仅注册 read/grep/glob/list_dir/diagnostics；可选 `web_fetch`（只读策略）。
- Runner **拒绝** write/shell/apply_patch/MCP 写。
- 输出仅 stdout/TUI，**不**自动 `write_file`。

### 7.5 子代理（Phase 6）

```go
// internal/agent/subagent/runner.go
type SubRunner struct {
    LLM   llm.Client
    Tools *tool.Registry // 只读子集
    Perm  *permission.Engine
}
```

- 主 Runner `task` 工具：并发池默认 3；返回摘要字符串。
- 回注：`role=tool`, `name=task`, `tool_call_id` 关联。

---

## 8. LLM 客户端（internal/llm）

### 8.1 抽象接口

```go
package llm

type Client interface {
    Chat(ctx context.Context, req Request) (*Response, error)
}

type Request struct {
    Model            string
    Messages         []Message
    Tools            []ToolDef
    MaxTokens        int
    Stream           bool
    ThinkingType     string
    ReasoningEffort  string
    UserID           string
    StrictTools      bool
}

type Response struct {
    Content           string
    ReasoningContent  string
    ToolCalls         []ToolCall
    FinishReason      string
    Usage             Usage
}

type Usage struct {
    PromptTokens          int
    CompletionTokens      int
    PromptCacheHitTokens  int
}
```

### 8.2 DeepSeek 实现要点

见 [llm-deepseek.md](llm-deepseek.md)：

- `stream_options.include_usage: true`
- `strict_tools` → `base_url` beta
- 流式末 chunk 解析 `usage`
- 429/5xx 指数退避

### 8.3 序列化共享

`internal/llm/deepseek/serialize.go`：

- `MergeSystem(...)` 
- `SerializeMessages([]Message)`
- `BuildToolsJSON([]ToolDef)`

`CountBreakdown` **必须**调用同一套序列化，保证与上屏一致。

---

## 9. 工具系统（internal/tool）

### 9.1 Registry

```go
type Registry struct {
    tools map[string]Tool
}

type Tool interface {
    Name() string
    Schema() ToolDef          // JSON Schema for API
    Execute(ctx context.Context, args json.RawMessage) (string, error)
    PermissionLevel() permission.Level
}
```

MCP 工具名：`mcp__{server}__{tool}`。

### 9.2 内置工具清单

| 工具 | Phase | Permission | 截断 |
|------|-------|------------|------|
| `read_file` | 1 | Low | 500 行 + max_chars |
| `grep` | 1 | Low | head_limit 200 |
| `shell` | 1 | Highest | max_chars |
| `glob` / `list_dir` | 1.5 | Low | 条数上限 |
| `apply_patch` | 2 | High | 变更行数上限 |
| `write_file` | 2 | High | — |
| `task` | 6 | Low | 摘要 4K 量级 |
| `web_fetch` / `web_search` | 6 | Medium | allowlist |
| `diagnostics` | 6 | Low | LSP 摘要 |

**不实现** `edit_file`；编辑统一 `apply_patch`（Codex 语义）。

### 9.3 apply_patch（Phase 2）

- 输入：unified diff 文本。
- 事务：解析 → 校验路径 → 备份/写入；失败**原子回滚**。
- 与 `permission.Engine` 校验路径与敏感文件。

### 9.4 工具结果包装（Prompt 安全）

```
<tool_result name="grep" id="call_abc123">
...body...
</tool_result>
```

---

## 10. 权限引擎（internal/permission）

### 10.1 模式

| 模式 | 行为 |
|------|------|
| `readonly` | 拒绝一切写/shell |
| `ask`（默认） | 写/shell/网络 弹 TUI 确认 |
| `auto` | `--dangerously-auto`；CI 可配置 |

### 10.2 Check 流程

```go
func (e *Engine) Check(tool string, args map[string]any) error {
    if e.mode == readonly && isWriteTool(tool) { return ErrDenied }
    if e.isSensitivePath(args) { return ErrDenied }
    if e.isHighRiskShell(args) { return e.prompt(...) }
    ...
}
```

- **MCP 写操作**与内置工具**同一** `Check` 入口（S6）。
- 路径：`filepath.Clean`、禁止 `..` 逃逸、symlink 解析（S2）。

---

## 11. 用户界面（internal/ui）

### 11.1 模式

| 模式 | 入口 |
|------|------|
| 交互 TUI | `ds-code`（Bubble Tea） |
| 非交互 | `ds-code -p "..."` / `--json` |
| Plan | `--plan` / `/plan` |

### 11.2 TUI 布局

```
┌─────────────────────────────────────┐
│ 对话区（流式 content + 折叠 reasoning）│
├─────────────────────────────────────┤
│ 工具日志（可选折叠）                    │
├─────────────────────────────────────┤
│ 输入框  [/ 补全]                       │
├─────────────────────────────────────┤
│ 状态栏  model·effort │ in·out·cache  │
└─────────────────────────────────────┘
```

### 11.3 Slash 命令

- **解析**：`internal/ui/slash.Parse` — 仅行首 `/`；正则见 PLAN。
- **注册表**：`registry.go` 单一数据源（help、补全、执行分发）。
- **未知命令**：TUI 提示；**不**写 messages、**不调** Agent。

### 11.4 /context 面板

**数据组装**：

```go
snap := session.UsageSnapshot(sess)
view, _ := context.BuildAPIContext(ctx, sessionID)
bd, _ := context.CountBreakdown(ctx, view)
panel.Render(snap, bd)
```

两层 UI：会话累计 + 六分项预估（见 PLAN §/context）。

### 11.5 取消

- `context.Context` 贯穿：LLM HTTP、tool Execute、shell 子进程、子代理。
- Ctrl+C → cancel → 停止流式读取 + `Process.Kill`。

---

## 12. 配置（internal/config）

配置文件：`~/.config/ds-code/config.yaml`，示例见 `configs/example.yaml`。

```yaml
llm:
  base_url: https://api.deepseek.com
  model: deepseek-v4-pro
  max_tokens: 16384
  strict_tools: false
  thinking:
    type: enabled
  reasoning_effort: max

context:
  window_tokens: 1048576
  compact_threshold_ratio: 0.80
  keep_recent_turns: 6
  truncate_by: chars
  tool_result_max_chars: 100000
  at_reference_max_chars: 128000

permission:
  mode: ask

btw:
  include_recent_turns: 0
  max_tokens: 4096
  count_toward_session: false

session:
  db_path: ~/.local/share/ds-code/sessions.db

non_interactive:
  ephemeral_session: true
```

环境变量覆盖：`DEEPSEEK_API_KEY`、`DS_CODE_CONFIG` 等（Viper）。

---

## 13. MCP（internal/mcp，Phase 5）

```go
type Manager struct {
    servers []ServerConfig
}

func (m *Manager) ListTools(ctx context.Context) ([]tool.Tool, error)
func (m *Manager) Call(ctx context.Context, name string, args json.RawMessage) (string, error)
```

- 崩溃隔离：单 server panic 不影响主进程。
- 工具名归一化后注册进 `tool.Registry`。

---

## 14. Checkpoint（internal/checkpoint，Phase 7）

- 写操作前：记录 affected files 的 hash 或 patch 集。
- `/rewind n`：回滚工作区；`messages` **追加** system 事件行。
- 首期可仅记录 patch 路径，不全量快照仓库。

---

## 15. 安全设计

### 15.1 审计清单映射

| ID | 设计落点 |
|----|----------|
| S1 | config + env；日志 redact |
| S2 | `permission.resolvePath` |
| S3 | `sensitivePatterns` denylist |
| S4 | shell 高危 + cancel kill |
| S5 | tool_result 边界；system 固定序 |
| S6 | MCP → `Perm.Check` |
| S7 | session_id 隔离；DB 文件权限 0600 |
| S8 | CI govulncheck |
| S9 | context 贯穿 |
| S10 | `--audit-log` JSONL |
| S11 | TruncateToolResult / @ max_chars |
| S12 | compact prompt 脱敏 |
| S13 | btw 无 tools、不写 DB |

### 15.2 威胁简要

| 威胁 | 缓解 |
|------|------|
| Prompt 注入 via tool | 边界标签 + system 不可覆盖 |
| 路径遍历 | Clean + jail 到 workspace |
| SSRF（web） | allowlist、默认关 |
| API Key 泄露 | 禁入 repo、日志脱敏 |

---

## 16. 可观测性与错误

| 类型 | 策略 |
|------|------|
| 日志 | slog；debug 可 dump request id，不 dump key |
| 指标 | Phase 7+ 可选：turn 数、tool 失败率 |
| 用户错误 | API 4xx/5xx → 友好文案 + 是否可重试 |
| compact 失败 | 降级截断 + 横幅警告 |

---

## 17. 测试策略

| 层级 | 范围 |
|------|------|
| 单元 | `slash.Parse`、`mergeSystem`、`CountBreakdown` 不变式、`SessionUsedTokens` |
| 集成 | mock `llm.Client` 驱动 Runner 多轮 tool |
| 契约 | serialize 与 tokenizer Count 一致性（有 CGO 时） |
| E2E | `ds-code -p` 对样例仓库（需 API Key，CI 可选 nightly） |

---

## 18. 分阶段交付映射

| Phase | 设计章节 | 交付物 |
|-------|----------|--------|
| 0 | §12、目录 | cobra、config、CI |
| 1 | §6–§8、§9.2 部分 | Runner、PrepareRequest、read/grep/shell |
| 1.5 | §6.6、§11.3 | @、slash 骨架、git |
| 2 | §9.3、§10 | apply_patch、permission、strict |
| 3 | §5、§6.4 | SQLite、compact、resume、clear |
| 4 | §11 | TUI、/context、状态栏 |
| 5 | §13 | MCP |
| 6 | §6.6、§7.5、§9.2 | 子代理、Rules、Skills、Plan、LSP、Web |
| 7 | §14、§15 | checkpoint、审计加固 |

---

## 19. 开放问题（实现时决议）

| # | 问题 | 建议默认 |
|---|------|----------|
| O1 | `/mode` 作用域 global vs per-session | **per-session**（PLAN 建议） |
| O2 | compact 后 usage 仍 >80% 是否二次 compact | 单次 Prepare 最多 compact 一次；仍超限靠 API 重试 |
| O3 | `CharCounter` 与 tokenizer 并存时的测试矩阵 | CI 无 CGO 跑 CharCounter 路径 |
| O4 | 费用估算数据源 | Phase 4 静态单价表 + usage |

---

## 20. 文档索引

| 文档 | 内容 |
|------|------|
| [PLAN.md](PLAN.md) | 路线图、对标、Slash 表、验收 |
| [llm-deepseek.md](llm-deepseek.md) | API 字段、思考模式、usage、strict |
| **DESIGN.md**（本文） | 模块、Schema、流程、接口 |

---

## 变更记录

| 日期 | 版本 | 摘要 |
|------|------|------|
| 2026-05-16 | v1.0 | 基于 PLAN v0.11 首版详细设计 |
