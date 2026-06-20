# v0.1.4 需求文档

> 版本：v0.1.4  
> 状态：规划中  
> 更新日期：2026-06-21

## 1. 目标

1. **内建工具提示词全面改写（P0 核心）**：对 registry 中全部内建工具的 `Description()` 与 JSON Schema 字段 `description` 重写一遍，风格与边界由 **你** 审定（见 FR-1、[TOOL_PROMPTS.md](TOOL_PROMPTS.md)）。
2. **协作流程**：实现者/Agent 在改写任一工具提示词前须先提问或提交草稿，**未经你确认不得合入定稿**（见 §2）。
3. **工具名与引用一致（P0 配套）**：LLM 可见 shell 工具名改为 `bash`；提示词中引用其它工具时用 `tool.Name*` 注入，避免硬编码（见 FR-2）。
4. **系统提示词（P1，你主导）**：`prompt.md` 若在本版合入，正文由你提供或逐段批准；代码侧仅负责 embed + 模板注入（见 FR-3）。

**非目标**：工具执行逻辑变更；权限 S2/S3 变更；MCP 工具 description；子代理 `PromptOverlay`；历史 `shell` tool_call 别名；未经你确认的提示词大批量替换。

## 2. 协作流程（必遵）

```mermaid
flowchart LR
  A[你指定工具与风格] --> B[实现者出草稿/选项]
  B --> C{你确认?}
  C -->|否| B
  C -->|是| D[落代码 text.go]
  D --> E[更新 TOOL_PROMPTS.md 状态]
  E --> F[单测 + ACCEPTANCE 勾选]
```

| 步骤 | 责任 | 说明 |
|------|------|------|
| 1. 排期 | 你 | 在 [TOOL_PROMPTS.md](TOOL_PROMPTS.md) 标明优先工具（建议每次 1～3 个） |
| 2. 草稿 | 实现者 | 在对话/PR 中给出 Desc + 主要 Schema 文案，**不直接静默合入** |
| 3. 审定 | 你 | 确认、修改或否决 |
| 4. 实现 | 实现者 | 仅写入已审定文案；更新台账状态为「已定稿」 |
| 5. 回归 | 实现者 | `make test`；必要时补 description 相关断言 |

**禁止**：参照 Cursor/Claude Code 长文擅自扩写全套工具描述；未经你确认将「规划中」文案标为已定稿。

## 3. 用户故事

### US-1：每个工具都有清晰、一致的 LLM 指引

**作为** ds-code 维护者（你），  
**我希望** 每个内建工具的 description 明确用途、边界与和其它工具的分工，  
**以便** 模型少绕行 `bash`、少误用 `write_file` 代替 `apply_patch`。

**验收**：[TOOL_PROMPTS.md](TOOL_PROMPTS.md) 12 项均为「已定稿」；ACCEPTANCE §3 逐工具通过。

### US-2：改写过程可控

**作为** 提示词作者，  
**我希望** 实现者在改文案前先问我，  
**以便** 提示词反映我的产品判断，而不是模型的默认习惯。

**验收**：每个已定稿工具在台账有确认记录（PR 讨论或对话引用）。

### US-3：工具名改名不再次撕裂文案

**作为** 维护者，  
**我希望** `bash` 改名后 grep/read_file 等描述自动引用正确 wire 名，  
**以便** 下次改名不必全文搜索。

**验收**：FR-2；grep/bash 相关单测绿。

## 4. 功能需求

### FR-1 内建工具提示词全面改写（核心）

覆盖工具（与 [TOOL_PROMPTS.md](TOOL_PROMPTS.md) 一致）：

`read_file`、`grep`、`glob`、`list_dir`、`diagnostics`、`web_fetch`、`web_search`、`bash`、`apply_patch`、`write_file`、`tool_search`、`agent`，以及共享 `builtin/text.go` 中的 `Schema*` 常量。

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-1.1 | 每个已注册工具的 `Description()` 文案完成审定并合入 | P0 |
| FR-1.2 | 每个工具 JSON Schema 的 `properties.*.description` 完成审定并合入 | P0 |
| FR-1.3 | `tool_search` 文案从 `tool_search.go` 抽到 `text.go`（与其它工具一致） | P1 |
| FR-1.4 | 探索类工具（grep/glob/list_dir/diagnostics）对 `.git`、`skip_dirs`、先收窄 `path` 的说明风格与你确认后统一 | P1 |
| FR-1.5 | 写工具（apply_patch/write_file/bash）对分工、高危操作的说明与你确认后统一 | P1 |
| FR-1.6 | `agent` 对四种类型的说明与你确认；是否与 spawn overlay 去重由你决定 | P1 |
| FR-1.7 | `web_search` 占位描述可与实现状态一致（未注册时仍维护文案供后续启用） | P2 |
| FR-1.8 | 各工具 `*.md` 设计文档与定稿 Desc 同步（可滞后于 text.go） | P2 |

#### FR-1.9 单工具交付定义（Done）

- [ ] `text.go`（或等价）中 Desc + Schema 常量已更新  
- [ ] 你已在台账或 PR 中确认  
- [ ] 若含跨工具引用，使用 `tool.Name*` 注入而非硬编码（除非你为可读性明确要求字面量）  
- [ ] `make test` 通过（含既有 description 快照测试，若有）

### FR-2 工具名 `bash` 与引用注入（配套）

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-2.1 | `internal/toolname.Bash = "bash"`；`tool.NameShell` 指向之 | P0 |
| FR-2.2 | permission / runner / display / hooks 用 `NameShell.Matches` 或 `toolname.Bash` | P0 |
| FR-2.3 | `tools.defer_builtin` 文档与示例使用 `bash` | P0 |
| FR-2.4 | 工具 Desc 中引用终端工具时写「bash」或通过注入生成 | P0 |
| FR-2.5 | YAML 键 `tools.shell` **不改** | P0 |

### FR-3 系统提示词（次要，你主导）

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-3.1 | `prompt.md` 载体（embed + template）可按你已审定内容合入 | P1 |
| FR-3.2 | 模板变量注入 `Bash`、`ReadFile` 等，与 FR-1 工具名一致 | P1 |
| FR-3.3 | 系统提示词正文章节与措辞 **全部由你审定**；未审定部分不得标为 v0.1.4 完成 | P0 |
| FR-3.4 | 系统提示词与工具 Desc 重复的内容，以你决定保留在哪一层（system vs tool）为准 | P1 |

### FR-4 测试与文档

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-4.1 | [TOOL_PROMPTS.md](TOOL_PROMPTS.md) 随进度更新 | P0 |
| FR-4.2 | [ACCEPTANCE.md](ACCEPTANCE.md) 逐工具勾选 | P0 |
| FR-4.3 | `CHANGELOG.md` 记录 `bash` 改名 breaking | P0 |
| FR-4.4 | CONFIG / builtin README defer 示例更新 | P1 |

## 5. 非功能需求

| ID | 描述 |
|----|------|
| NFR-1 | `make test` / `make lint` / `make vet` 全绿 |
| NFR-2 | 单工具 Desc 长度与你确认的风格一致；不为了省 token 擅自砍你定的长文 |
| NFR-3 | 提示词仅存在于 `text.go` / `prompt.md`；业务逻辑 `.go` 不夹带大段 LLM 文案 |
| NFR-4 | 改写不改变 `Schema()` 字段名、类型、enum（除非另开需求） |

## 6. 范围边界

**In scope**

- `internal/tool/builtin/**/text.go`、`builtin/text.go`
- `internal/tool/builtin/tool_search/`（抽 text.go）
- `internal/toolname/**`、`NameShell` 改名链
- `internal/prompt/**`（仅你已审定的 system 内容）
- `docs/v0.1.4/**`

**Out of scope**

- MCP adapter `Description()`
- `agentdef.PromptOverlay`
- 工具行为、权限、TUI
- 自动迁移用户 `defer_builtin`

## 7. 实现优先级建议

| 阶段 | 内容 | gate |
|------|------|-------|
| **0** | 已确认：长文风格、全工具覆盖、system 逐段审定（见 [TOOL_PROMPTS.md](TOOL_PROMPTS.md)） | 已完成 |
| **A** | `bash` 改名 + 注入基础设施 | 可并行 |
| **B** | **全部**内建工具：草稿 → 你确认 → `text.go`（建议一次 1 个工具） | 每工具须你确认 |
| **C** | 共享 `builtin/text.go` Schema 统一审定 | 你确认 |
| **D** | 系统 `prompt.md` 逐段审定合入 | 你逐段确认 |
| **E** | CHANGELOG、CONFIG、ACCEPTANCE 收尾 | — |
