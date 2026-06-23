# v0.1.4 需求文档

> 版本：v0.1.4  
> 状态：规划中  
> 更新日期：2026-06-23  

## 1. 目标

1. **内建工具提示词全面改写（P0 核心）**：对 registry 中全部内建工具的 `Description()` 与 JSON Schema 字段 `description` 重写一遍，风格与边界由 **你** 审定（见 FR-1、[TOOL_PROMPTS.md](TOOL_PROMPTS.md)）。
2. **协作流程**：实现者/Agent 在改写任一工具提示词前须先提问或提交草稿，**未经你确认不得合入定稿**（见 §2）。
3. **工具名与引用一致（P0 配套）**：LLM 可见 shell 工具名改为 `bash`；提示词中引用其它工具时用 `tool.Name*` 注入，避免硬编码（见 FR-2）。
4. **系统提示词（P1，你主导）**：`prompt.md` 若在本版合入，正文由你提供或逐段批准；代码侧仅负责 embed + 模板注入（见 FR-3）。

**非目标**：权限 S2/S3 变更；MCP 工具 description；子代理 `PromptOverlay`；历史 `shell` tool_call 别名；未经你确认的提示词大批量替换。

**本版 P0 例外**：`bash` 工具行为变更（`timeout_ms`、`run_in_background`、移除 `list_jobs`、超时 kill、TUI 倒计时）见 [FR-5](REQUIREMENTS.md#fr-5-bash-工具行为p0)；`grep` 后端改为 ripgrep 见 [FR-6](REQUIREMENTS.md#fr-6-grep-工具-ripgrep-后端p0)。

## 2. 协作流程（必遵）

```mermaid
flowchart LR
  A[你指定工具与风格] --> B[实现者出草稿/选项]
  B --> C{你确认?}
  C -->|否| B
  C -->|是| D[落代码 usage.prompt + text.go]
  D --> E[更新 TOOL_PROMPTS.md 状态]
  E --> F[单测 + ACCEPTANCE 勾选]
```

| 步骤 | 责任 | 说明 |
|------|------|------|
| 1. 排期 | 你 | 在 [TOOL_PROMPTS.md](TOOL_PROMPTS.md) 标明优先工具（建议每次 1～3 个） |
| 2. 草稿 | 实现者 | 在对话/PR 中给出 Desc + 主要 Schema 文案，**不直接静默合入** |
| 3. 审定 | 你 | 确认、修改或否决 |
| 4. 实现 | 实现者 | 按 [FR-0](#fr-0-工具-prompt-标准模式必遵) 写入已审定文案；更新台账为「已定稿」 |
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

### FR-0 工具 prompt 标准模式（必遵）

后续 **全部** 内建工具 Description 改写须与 [`bash` 参考实现](../../internal/tool/builtin/shell/) 一致：**`usage.prompt` + `//go:embed` + `text/template`**。（系统层仍用 [`internal/prompt/prompt.md`](../../internal/prompt/prompt.md)。）

#### 目录与职责

```text
internal/tool/builtin/<tool>/
├── usage.prompt   # Description 正文（Markdown）；跨工具引用用 {{.ReadFile}} 等
├── text.go        # embed、模板变量、RenderDesc()、Schema*、Err*、Result*
├── <tool>.go      # Description() { return RenderDesc() }
└── text_test.go   # 建议：断言工具名已注入、无 {{. 残留
```

| 文件 | 内容 | 禁止 |
|------|------|------|
| `usage.prompt` | 发给 LLM 的 `tools[].description` 正文 | 硬编码 wire 工具名（须用模板占位符） |
| `text.go` | `//go:embed usage.prompt`；`descVars`（或 `<tool>Vars`）；`RenderDesc()`；JSON Schema 字段 `description` 常量 | 大段 Description 字符串、`fmt.Sprintf` 拼正文 |
| `<tool>.go` | `Description()` 委托 `RenderDesc()` | 内联长文案 |

#### FR-0 需求条目

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-0.1 | 每个工具子包新增或迁移至 `usage.prompt` 承载 Description 正文 | P0 |
| FR-0.2 | `text.go` 使用 `//go:embed usage.prompt` + `text/template` 渲染 | P0 |
| FR-0.3 | 模板变量值来自 `tool.Name*`（或 `toolname` 包）；`usage.prompt` 中用 `{{.Field}}` 引用 | P0 |
| FR-0.4 | 导出 `RenderDesc() string`（命名可一致）；`<tool>.go` 的 `Description()` 仅调用之 | P0 |
| FR-0.5 | 模板渲染失败 `panic`（与 system prompt 一致，启动/测试即暴露） | P0 |
| FR-0.6 | 无需交叉引用的工具仍使用 `usage.prompt`（可为短文），**不得**退回 `const Desc*` 单文件常量 | P0 |
| FR-0.7 | 共享 `builtin/text.go` 仅保留跨工具 Schema 常量，**不**引入 embed | P0 |
| FR-0.8 | 参考实现：[`shell/usage.prompt`](../../internal/tool/builtin/shell/usage.prompt)、[`shell/text.go`](../../internal/tool/builtin/shell/text.go) | — |

#### 模板变量约定

- 字段名 PascalCase，与 `internal/prompt/text.go` 的 `systemBaseVars` 对齐（如 `Bash`、`ReadFile`、`Grep`、`Glob`、`ListDir`、`ApplyPatch`、`WriteFile`）。
- 各工具 `descVars` **只声明本工具 `usage.prompt` 用到的字段**；未用到的不必注入。
- 新增占位符时同步更新 `defaultDescVars()`（或等价函数）与单测。

### FR-1 内建工具提示词全面改写（核心）

覆盖工具（与 [TOOL_PROMPTS.md](TOOL_PROMPTS.md) 一致）：

`read_file`、`grep`、`glob`、`list_dir`、`diagnostics`、`web_fetch`、`web_search`、`bash`、`apply_patch`、`write_file`、`tool_search`、`agent`，以及共享 `builtin/text.go` 中的 `Schema*` 常量。

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-1.1 | 每个已注册工具的 `Description()` 文案完成审定并合入 | P0 |
| FR-1.2 | 每个工具 JSON Schema 的 `properties.*.description` 完成审定并合入 | P0 |
| FR-1.3 | `tool_search` 迁至 `usage.prompt` + `text.go`（FR-0），不再在 `.go` 内联 Description | P1 |
| FR-1.4 | 探索类工具（grep/glob/list_dir/diagnostics）对 `.git`、`skip_dirs`、先收窄 `path` 的说明风格与你确认后统一 | P1 |
| FR-1.5 | 写工具（apply_patch/write_file/bash）对分工、高危操作的说明与你确认后统一 | P1 |
| FR-1.6 | `agent` 对四种类型的说明与你确认；是否与 spawn overlay 去重由你决定 | P1 |
| FR-1.7 | `web_search` 占位描述可与实现状态一致（未注册时仍维护文案供后续启用） | P2 |
| FR-1.8 | 各工具 `*.md` 设计文档与定稿 Desc 同步（可滞后于 text.go） | P2 |

#### FR-1.9 单工具交付定义（Done）

- [ ] 符合 [FR-0](#fr-0-工具-prompt-标准模式必遵)：`usage.prompt` + `RenderDesc()` + embed/template  
- [ ] `text.go` 中 Schema 常量已更新（若本工具涉及）  
- [ ] 你已在台账或 PR 中确认  
- [ ] 交叉引用经 `{{.…}}` 注入，无硬编码 wire 名（除非你明确要求字面量）  
- [ ] `text_test.go`（建议）覆盖模板渲染；`make test` 通过  
- [ ] 旧 `const Desc*` / `fmt.Sprintf(Desc*, …)` 已删除

### FR-2 工具名 `bash` 与引用注入（配套）

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-2.1 | `internal/toolname.Bash = "bash"`；`tool.NameShell` 指向之 | P0 |
| FR-2.2 | permission / runner / display / hooks 用 `NameShell.Matches` 或 `toolname.Bash` | P0 |
| FR-2.3 | `tools.defer_builtin` 文档与示例使用 `bash` | P0 |
| FR-2.4 | 工具 Desc 中引用终端工具时写「bash」或通过注入生成 | P0 |
| FR-2.5 | YAML 键 `tools.shell` **不改** | P0 |
| FR-2.6 | Breaking：`background` → `run_in_background`；移除 `list_jobs` | P0 |

### FR-3 系统提示词（次要，你主导）

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-3.1 | 系统层 [`internal/prompt/prompt.md`](../../internal/prompt/prompt.md) 与工具层同模式（embed + template） | P1 |
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
| FR-4.5 | `grep` ripgrep：`grep.md`、`CHANGELOG`、Makefile `fetch-ripgrep` | P0 |

### FR-5 bash 工具行为（P0）

| ID | 描述 |
|----|------|
| FR-5.1 | Schema 新增 `timeout_ms`（sync 与 `run_in_background` 均适用，cap 600000ms） |
| FR-5.2 | `background` 改名为 **`run_in_background`**，默认 false；阻塞至完成，同轮可并行 |
| FR-5.3 | 移除 LLM 可见 **`list_jobs`**、`job_id`、`cancel` |
| FR-5.4 | 超时到期 **强制 kill** 子进程（sync 与 bg） |
| FR-5.5 | TUI：sync 与 `run_in_background` bash Running 标题 **倒计时** |
| FR-5.6 | 退出 ds-code 时 kill 本会话 running job；不跨会话恢复（`reconcileStaleJobs`） |
| FR-5.7 | Breaking 写入 CHANGELOG；无 `background`/`list_jobs` 别名 |

### FR-6 grep 工具 ripgrep 后端（P0）

| ID | 描述 |
|----|------|
| FR-6.1 | 后端改为 ripgrep 15.1.0；`bundled`（embed tar.gz → `~/.ds-code/bin/rg`）/ `system` / `path` |
| FR-6.2 | Schema 对齐 Claude Code：`glob`、`-B/-A/-C`、`head_limit`/`offset`、`multiline` 等 |
| FR-6.3 | 输出格式：`Found N files` / `path:line:text` / `Found X occurrences across Y files` + 分页脚标 |
| FR-6.4 | `path` 与 `glob` 分离；正则方言 Rust/ripgrep |
| FR-6.5 | `tools.grep` 新增 `timeout`、`binary`、`binary_path`、`respect_gitignore` |
| FR-6.6 | 不搜索 `.git`（宽泛 `path` 加 `!.git/**`；`path=.git` 空结果） |
| FR-6.7 | FR-0：`usage.prompt` + 中文 Schema + `{{.Grep}}/{{.Bash}}/{{.Agent}}` 注入 |
| FR-6.8 | `make fetch-ripgrep`；`go test ./internal/tool/builtin/grep/...` 全绿 |

## 5. 非功能需求

| ID | 描述 |
|----|------|
| NFR-1 | `make test` / `make lint` / `make vet` 全绿 |
| NFR-2 | 单工具 Desc 长度与你确认的风格一致；不为了省 token 擅自砍你定的长文 |
| NFR-3 | Description 正文仅在各工具 `usage.prompt` 与 system `internal/prompt/prompt.md`；Schema/Err/Result 在 `text.go`；业务 `<tool>.go` 不夹带 LLM 文案 |
| NFR-4 | 改写不改变 `Schema()` 字段名、类型、enum（除非另开需求） |

## 6. 范围边界

**In scope**

- `internal/tool/builtin/**/usage.prompt`、`**/text.go`、`**/text_test.go`
- `internal/tool/builtin/tool_search/`（FR-0 迁移）
- `internal/toolname/**`、`NameShell` 改名链
- `internal/prompt/**`（仅你已审定的 system 内容）
- `docs/v0.1.4/**`

**Out of scope**

- MCP adapter `Description()`
- `agentdef.PromptOverlay`
- 工具行为、权限、TUI（**`bash` FR-5、`grep` FR-6 除外**）
- 自动迁移用户 `defer_builtin`

## 7. 实现优先级建议

| 阶段 | 内容 | gate |
|------|------|-------|
| **0** | 已确认：长文风格、全工具覆盖、system 逐段审定（见 [TOOL_PROMPTS.md](TOOL_PROMPTS.md)） | 已完成 |
| **A** | `bash` 改名 + FR-0 参考实现（`shell/usage.prompt`） | 已完成参考 |
| **B** | **全部**工具：草稿 → 你确认 → `usage.prompt` + `text.go`（一次 1 个） | 每工具须 FR-0 + 你确认 |
| **C** | 共享 `builtin/text.go` Schema 统一审定 | 你确认 |
| **D** | 系统 `prompt.md` 逐段审定合入 | 你逐段确认 |
| **E** | CHANGELOG、CONFIG、ACCEPTANCE 收尾 | — |
