# ds-code 配置参考

> 文档版本：v1.2  
> 更新日期：2026-05-16  
> 状态：实现基线（与 [PLAN.md](PLAN.md) v0.13+ 对齐）  
> 相关：[DESIGN.md](DESIGN.md)（模块实现）、[llm-deepseek.md](llm-deepseek.md)（API 字段与模型契约）

本文是 **配置项 YAML 键**、**CLI 参数**、**环境变量** 的权威清单。其它文档仅摘要并链接至此。

---

## 1. 配置来源与优先级

按**叶子键**深度合并，**后读覆盖先读**：

```text
内置默认值 → 用户级 YAML → 项目级 YAML → CLI flags
              （最低）                         （最高）
```

| 优先级 | 来源 | 路径 / 机制 |
|--------|------|-------------|
| 1（低） | 用户级 | `~/.ds-code/config/config.yaml` |
| 2 | 项目级 | `<git-root>/.ds-code/config.yaml`（自 **cwd** 向上至 git 根） |
| 3（高） | CLI | `ds-code` 全局与子命令 flags（仅当前进程） |

- 未在高层出现的键**保留**低层已加载值；**非**整文件替换。
- **`llm.api_key`**：**不参与**上述 YAML/CLI 合并链，**仅**由环境变量注入（见 §3.1）；配置文件与 CLI **不得**设置。
- **用户数据根** `~/.ds-code/`：**固定路径**，不可通过环境变量 / CLI / YAML 修改（见 §2）。
- **环境变量** `DS_CODE_*`（除 API Key 外）：可选，经 Viper 绑定 YAML 叶子键（见 §3.2）；仍被项目级 YAML 与 CLI 覆盖。

**实现约定**（`internal/config`）：`SetDefault` → 读用户 YAML → 读项目 YAML → `BindPFlags` → 解析 `project_root` → 创建 `projects/<project_id>/` → `LoadAPIKey()`。

---

## 2. 用户数据目录（`~/.ds-code/`，固定路径）

本机**全局**状态与**项目内** `.ds-code/`（仓库根）分离；**运行时数据按项目分目录**存放。

**用户数据根目录**恒为当前用户主目录下的 **`~/.ds-code/`**（`$HOME/.ds-code`）。**不支持** `DS_CODE_HOME`、`XDG_*` 重定向，**无** `--data-dir` / `--config` 类 CLI。实现：`filepath.Join(os.UserHomeDir(), ".ds-code")`。

```text
~/.ds-code/
├── config/
│   └── config.yaml           # 用户级配置（§5）
├── skills/                   # 用户级 Skills（可选）
│   └── <name>/SKILL.md
└── projects/
    └── <project_id>/         # 每个打开的项目一份（§2.1）
        ├── project.meta.json # 可选：记录项目根绝对路径，便于排查
        ├── sessions.db       # 默认 SQLite
        ├── audit.jsonl       # 启用审计时写入（路径固定）
        └── checkpoints/      # Phase 7：检查点（路径固定）
```

| 路径 | 说明 |
|------|------|
| `~/.ds-code/config/config.yaml` | 用户级 YAML（**全项目共享**） |
| `~/.ds-code/skills/` | 与 `<git-root>/.ds-code/skills/` 并列 |
| `~/.ds-code/projects/<project_id>/` | **当前项目**的运行时数据根（DB 等） |

目录权限：`~/.ds-code/`、`config/`、`projects/` 建议 **0700**；`sessions.db` 等文件 **0600**。

完整示例 YAML 见 [`configs/example.yaml`](../configs/example.yaml)。

### 2.1 项目运行时目录（`projects/<project_id>/`）

每个**项目根目录**对应 `~/.ds-code/projects/` 下的一个子文件夹，**互不共享** SQLite / 审计 / checkpoint 等运行时文件。

#### 项目根（`project_root`）

与 `AGENTS.md`、项目级 `config.yaml` 解析一致：

1. 自 **cwd** 向上查找 **git 根**（含 `.git` 的目录）；
2. 若未找到 git 仓库，则 `project_root` = **cwd** 经 `filepath.Abs` + `filepath.Clean` 后的绝对路径；
3. 建议对 `project_root` 再执行 `filepath.EvalSymlinks`，避免符号链接导致同一仓库产生多个 `project_id`。

#### `project_id`（目录名）

```text
project_id = hex(SHA256([]byte(project_root)))
```

- 哈希输入：**项目根的绝对路径字符串**（UTF-8，与 Go `sha256.Sum256` 一致）；
- 输出：**小写**十六进制，64 字符，无 `-` 前缀；
- 示例：`/Users/alice/code/ds-code` → `projects/a1b2c3.../`（以实际哈希为准）。

#### 项目目录内文件（固定路径，均不可配置）

`~/.ds-code/projects/<project_id>/` 下的运行时文件**路径全部由实现写死**，不支持 YAML、环境变量或 CLI 指定路径。

| 文件 / 目录 | 固定路径 | 说明 |
|-------------|----------|------|
| SQLite | `sessions.db` | 会话与 `messages` 历史 |
| 审计日志 | `audit.jsonl` | 启用审计时追加 JSONL（tool 名 + 参数哈希） |
| 检查点 | `checkpoints/` | Phase 7；patch / 文件哈希 |

启动时：`project_root` → `project_id` → `MkdirAll` 项目目录 → 按上表打开/创建文件。

**禁止**在配置文件中出现路径类键（拒绝启动）：`session`、`session.db_path`、`audit.log_path`、`checkpoint.*` 等。仅允许 [§5.5](#55-audit--开关phase-2) 的 **`audit.enabled`** 开关。

#### `project.meta.json`（推荐）

创建项目目录时写入，便于 `ls ~/.ds-code/projects/` 时辨认：

```json
{
  "root": "/Users/alice/code/my-app",
  "created_at": "2026-05-16T12:00:00Z"
}
```

**实现**（`internal/config` 或 `internal/session`）：

```go
func UserDataHome() string
func ProjectID(projectRoot string) string
func ProjectDataDir(projectRoot string) (string, error)
func DefaultDBPath(projectRoot string) string
func DefaultAuditLogPath(projectRoot string) string   // …/audit.jsonl
func DefaultCheckpointDir(projectRoot string) string // …/checkpoints/
```

---

## 3. 环境变量

用户级配置文件路径固定为 **`~/.ds-code/config/config.yaml`**，不可通过环境变量或 CLI 指定。

### 3.1 DeepSeek API Key（`llm.api_key`）

运行时字段 `llm.api_key` **只能**从环境变量读取，**不能**写在 YAML，**没有** `--api-key` CLI。

解析顺序（先非空者生效）：

```text
DS_CODE_DEEPSEEK_API_KEY → DEEPSEEK_API_KEY → 报错退出
```

| 变量 | 说明 |
|------|------|
| `DS_CODE_DEEPSEEK_API_KEY` | **首选**；ds-code 专用，避免与其它工具共用 `DEEPSEEK_API_KEY` 冲突 |
| `DEEPSEEK_API_KEY` | 回退；兼容官方文档与现有脚本 |

两者均未设置或为空时：`internal/config` **必须**返回明确错误（如 `missing DeepSeek API key: set DS_CODE_DEEPSEEK_API_KEY or DEEPSEEK_API_KEY`），**不得**静默继续。

实现：`LoadAPIKey()` 在合并 YAML 之后执行；YAML 若含 `llm.api_key` 键，**拒绝启动**（防止误写入配置文件）。

### 3.2 Viper 自动绑定（可选实现）

设置 `AutomaticEnv` + 前缀 `DS_CODE_` 时，环境变量可覆盖**与用户级 YAML 同级**的键（仍低于项目级 / CLI）：

| 环境变量 | YAML 键 |
|----------|---------|
| `DS_CODE_LLM_MODEL` | `llm.model` |
| `DS_CODE_LLM_MAX_TOKENS` | `llm.max_tokens` |
| `DS_CODE_PERMISSION_MODE` | `permission.mode` |

规则：`.` → `_`，全大写。未列出的键按同一规则类推（**不含** `session.*`、**不含**数据根路径；见 §2）。

---

## 4. CLI 参数

基于 `cobra`；**全局 persistent flags** 在子命令间共享。下表「配置键」表示覆盖的 YAML 叶子路径。

### 4.1 全局 flags

| Flag | 短选项 | 配置键 | 类型 | 说明 |
|------|--------|--------|------|------|
| `--model` | — | `llm.model` | string | `deepseek-v4-pro` \| `deepseek-v4-flash` |
| `--max-tokens` | — | `llm.max_tokens` | int | 单次 completion 上限，≤ `393216` |
| `--thinking` | — | `llm.thinking.type` | string | `enabled` \| `disabled` |
| `--reasoning-effort` | — | `llm.reasoning_effort` | string | `high` \| `max` |
| `--strict-tools` | — | `llm.strict_tools` | bool | `true` → 全部请求走 Beta `base_url` |
| `--permission-mode` | — | `permission.mode` | string | `readonly` \| `ask` \| `auto` |
| `--dangerously-auto` | — | `permission.mode` | bool | 设为 `auto`；非 TTY 脚本用，文档警示 |
| `--plan` | — | `run_mode`（session） | bool | Plan 模式：只读工具子集 |
| `--audit-log` | — | `audit.enabled` | bool | 启用审计；写入固定路径 `…/audit.jsonl`（**非**路径参数） |
| `-p` | `--prompt` | — | string | **非交互**：单次 prompt 后退出 |
| `--json` | — | — | bool | 非交互输出 JSON（常配合 `-p`） |

> API Key **无** CLI flag；见 §3.1。

### 4.2 子命令

| 命令 | 说明 |
|------|------|
| `ds-code` | 无子命令时启动 **交互 TUI** |
| `ds-code -p "..."` | 非交互单次任务 |
| `ds-code --json -p "..."` | CI / 脚本；结构化输出 |
| `ds-code sessions` | 列出 SQLite 中的 session |
| `ds-code resume <id>` | 恢复指定 session |

子命令专属 flags 在实现时追加于本章并同步 `configs/example.yaml` 注释。

### 4.3 与 TUI / session 的关系

以下**不**走 CLI 全局 flag，写入 **SQLite `sessions` 表** 或仅当次有效：

| 机制 | 影响字段 | 说明 |
|------|----------|------|
| TUI `/mode` | `sessions.model` | 覆盖配置默认模型（建议 per-session） |
| TUI `/effort` | `sessions.reasoning_effort` | `high` \| `max` |
| TUI `/thinking on\|off` | `sessions.thinking_type` | 思考模式开关 |
| TUI `/plan` | `sessions.run_mode` | `agent` \| `plan` |
| 启动 `--plan` | 新 session 初始 `run_mode` | 与配置 `run_mode` 二选一实现时取 CLI |

---

## 5. YAML 配置项

类型与默认值供 `internal/config` 实现与校验；**加粗**为推荐默认。

### 5.1 `llm` — DeepSeek 客户端

| 键 | 类型 | 默认 | CLI | 说明 |
|----|------|------|-----|------|
| `llm.api_key` | string | — | — | **仅环境变量**（§3.1）；**禁止**出现在 YAML |
| `llm.base_url` | string | `https://api.deepseek.com` | — | `strict_tools: true` 时实现改为 `.../beta` |
| `llm.model` | string | **`deepseek-v4-pro`** | `--model` | 白名单：`deepseek-v4-pro`、`deepseek-v4-flash` |
| `llm.max_tokens` | int | **16384** | `--max-tokens` | `min(值, 393216)` |
| `llm.timeout` | duration | `120s` | — | HTTP 超时 |
| `llm.strict_tools` | bool | **false** | `--strict-tools` | 见 [llm-deepseek.md · strict 模式](llm-deepseek.md#strict-模式beta) |
| `llm.thinking.type` | string | **`enabled`** | `--thinking` | `enabled` \| `disabled` |
| `llm.reasoning_effort` | string | **`max`** | `--reasoning-effort` | `high` \| `max`（`low`/`medium`/`xhigh` 映射见 llm-deepseek） |

### 5.2 `context` — 窗口、compact、截断

| 键 | 类型 | 默认 | 说明 |
|----|------|------|------|
| `context.window_tokens` | int | **1048576** | 1Mi；与 `limits.go` 一致 |
| `context.max_output_tokens` | int | **393216** | 384Ki 硬顶 |
| `context.compact_threshold_ratio` | float | **0.80** | 阈值 = `ratio × window`；compact 见 [PLAN · A/B/C](PLAN.md#会话-token计费累计-vs-compact-触发) |
| `context.keep_recent_turns` | int | **6** | compact 后保留最近 N **用户轮**全文 |
| `context.truncate_by` | string | **`chars`** | `chars` \| `tokenizer`；工具/`@` 截断 |
| `context.tool_result_max_chars` | int | **100000** | 单次 tool 返回字符上限 |
| `context.at_reference_max_chars` | int | **128000** | `@` 引用预加载总字符上限 |
| `context.git_snapshot_max_chars` | int | **16000** | `git status` + `diff --stat` 注入 system 上限 |
| `context.at_dir_max_files` | int | **50** | `@dir/` 最多预读文件数 |
| `context.at_dir_max_depth` | int | **4** | `@dir/` 最大目录深度 |

compact 触发：**A** `CountBreakdown.Total`、**B** `prompt_tokens_total`、**C** API 过长（见 [llm-deepseek.md](llm-deepseek.md)）；计费展示 `prompt+completion` **不单独**触发 compact。

### 5.3 `permission` — 工具沙箱

| 键 | 类型 | 默认 | CLI | 说明 |
|----|------|------|-----|------|
| `permission.mode` | string | **`ask`** | `--permission-mode`、`--dangerously-auto` | `readonly` \| `ask` \| `auto` |

用户级与项目级 YAML **均不得**将 `permission.mode` 设为 `auto`。`auto` **仅能**通过 CLI `--permission-mode auto` 或 `--dangerously-auto`（flag 须显式传入）启用；环境变量 `DS_CODE_PERMISSION_MODE=auto` 在无上述 CLI 时也会被拒绝。TUI 中 `/permissions auto` 需加 `--yes` 确认。

内置工具与 MCP 写操作均走 `permission.Engine`（[DESIGN.md §10](DESIGN.md#10-权限引擎internalpermission)）。

### 5.4 项目运行时数据（固定路径，非配置项）

落在 `~/.ds-code/projects/<project_id>/` 下的 **sessions.db / audit.jsonl / checkpoints/** 均见 [§2.1 表格](#项目目录内文件固定路径均不可配置)，**不可**通过配置改路径。`internal/session`、`internal/audit`、`internal/checkpoint` 仅调用 `config.Default*Path(projectRoot)`。

### 5.5 `audit` — 开关（Phase 2+）

仅控制**是否写入**固定文件 `audit.jsonl`，**不**配置路径。

| 键 | 类型 | 默认 | CLI | 说明 |
|----|------|------|-----|------|
| `audit.enabled` | bool | **false** | `--audit-log` | `true` 时向 `…/audit.jsonl` 追加 JSONL |

### 5.6 `btw` — `/btw` 旁路请求

| 键 | 类型 | 默认 | 说明 |
|----|------|------|------|
| `btw.include_recent_turns` | int | **0** | 带入主对话最近 N 轮；默认不带 |
| `btw.max_tokens` | int | **4096** | btw 单次 `max_tokens` |
| `btw.count_toward_session` | bool | **false** | 是否计入 session Token 累计 |

`cache_scope` 每次 `btw-{uuid}` → API `user_id`（见 [PLAN.md · /btw](PLAN.md#btw-快速提问不进入主对话)）。

### 5.7 `non_interactive` — `-p` / `--json`

| 键 | 类型 | 默认 | 说明 |
|----|------|------|------|
| `non_interactive.ephemeral_session` | bool | **true** | 每次 `-p` 使用新 `session_id`，不污染交互 session |

非 TTY 下 `permission.mode` 默认仍为 `ask`：**无法弹窗时拒绝**写/shell/网络写类 tool（返回错误，不阻塞）；脚本须 `--permission-mode readonly`、`auto` 或 `--dangerously-auto`。

### 5.8 `mcp` — MCP 子进程（Phase 5）

| 键 | 类型 | 默认 | 说明 |
|----|------|------|------|
| `mcp.servers` | []object | `[]` | 见 [DESIGN.md §13](DESIGN.md#13-mcpinternalmcpphase-5)；每项含 `name`、`command`、`args`、`env` |

### 5.9 `web` — 网络工具（Phase 6）

| 键 | 类型 | 默认 | 说明 |
|----|------|------|------|
| `web.fetch_enabled` | bool | **false** | `web_fetch` 默认关 |
| `web.search_enabled` | bool | **false** | `web_search` 默认关 |
| `web.allowlist` | []string | `[]` | 允许的主机名；防 SSRF |

### 5.10 `agent` — Runner

| 键 | 类型 | 默认 | 说明 |
|----|------|------|------|
| `agent.max_turns` | int | **25** | 单条用户消息内 **子轮次**上限（见 [PLAN · 术语](PLAN.md#术语用户轮子轮次turn)） |

### 5.11 `tools` — 内置工具约束

| 键 | 类型 | 默认 | 说明 |
|----|------|------|------|
| `tools.parallel_tool_calls` | bool | **false** | 同一 assistant 多条 `tool_calls` 是否并行执行 |
| `tools.read_file.max_lines` | int | **500** | 单次 `read_file` 默认行上限（`start`/`end` 闭区间亦受此限） |
| `tools.read_file.max_bytes` | int | **2097152** (2MiB) | 文件总大小上限；超限拒绝整次读取 |
| `tools.grep.head_limit` | int | **200** | `grep` 默认匹配条数上限 |
| `tools.glob.max_results` | int | **200** | `glob` / `list_dir` 结果条数上限 |
| `tools.apply_patch.max_changed_lines` | int | **2000** | 单 patch 允许变更行数 |
| `tools.shell.timeout` | duration | **120s** | `shell` 同步执行超时 |
| `tools.shell.env_blacklist` | []string | `[]` | 子进程环境变量名正则黑名单（与内置 secret 键名过滤为 OR）；作用于 `shell`、后台 job、MCP stdio |
| `tools.task.max_parallel` | int | **3** | 子代理 `task` 并发上限 |
| `tools.task.summary_max_chars` | int | **16000** | 子代理摘要字符上限（约 4K tokens 量级） |

### 5.12 `lsp` — Language Server（Phase 6）

运行时依赖用户本机安装的 language server（**不**随 `ds-code` 分发）。设计见 [DESIGN.md §9.5](DESIGN.md#95-lsp-子系统internallspphase-6)。

#### 5.12.1 全局开关与限额

| 键 | 类型 | 默认 | 说明 |
|----|------|------|------|
| `lsp.enabled` | bool | **true** | `false` 时 `diagnostics` 工具返回「LSP 已禁用」 |
| `lsp.idle_shutdown` | duration | **120s** | 无 LSP 请求后关闭子进程 |
| `lsp.diagnostics_timeout` | duration | **20s** | 单次 `diagnostics` 调用等待诊断上限 |
| `lsp.max_files_per_call` | int | **10** | 单次工具调用最多 `didOpen` 文件数 |
| `lsp.max_issues_per_file` | int | **20** | 每文件最多输出条数 |
| `lsp.warmup_on_start` | []string | `[]` | 启动 TUI 时仅 `initialize` 的 server ID，如 `["go"]` |

#### 5.12.2 `lsp.servers` — 按 ID 覆盖

`lsp.servers` 为 map：**key = server ID**（`go`、`typescript`、`cpp`、`java`…），value：

| 字段 | 类型 | 说明 |
|------|------|------|
| `command` | string | 可执行文件（PATH 或绝对路径） |
| `args` | []string | 参数；须含 stdio 模式（如 `serve`、`--stdio`） |
| `extensions` | []string | 扩展名列表，含 `.` |
| `env` | map[string]string | 追加环境变量（可选） |
| `disabled` | bool | `true` 时跳过该 server |

未配置的 ID 使用内置默认（见 DESIGN §9.5.3）；**java** 内置无默认 `command`，须用户填写。

#### 5.12.3 内置默认（摘要）

| ID | command | args | extensions |
|----|---------|------|------------|
| `go` | `gopls` | `["serve"]` | `.go` |
| `typescript` | `typescript-language-server` | `["--stdio"]` | `.ts`, `.tsx`, `.js`, `.jsx` |
| `cpp` | `clangd` | `[]` | `.c`, `.h`, `.cpp`, `.hpp`, `.cc`, `.cxx` |
| `java` | — | — | `.java`（须用户配置） |

#### 5.12.4 配置示例

```yaml
lsp:
  enabled: true
  idle_shutdown: 120s
  diagnostics_timeout: 20s
  max_files_per_call: 10
  max_issues_per_file: 20
  warmup_on_start: ["go"]
  servers:
    go:
      command: gopls
      args: ["serve"]
    typescript:
      command: typescript-language-server
      args: ["--stdio"]
    cpp:
      command: clangd
      args: []
    java:
      command: /opt/jdtls/bin/jdtls
      args: []
      env:
        JAVA_HOME: /usr/lib/jvm/java-21
```

---

## 6. 配置示例

### 6.1 用户级（`~/.ds-code/config/config.yaml`）

```yaml
llm:
  model: deepseek-v4-pro
  max_tokens: 16384
  thinking:
    type: enabled
  reasoning_effort: max

permission:
  mode: ask

context:
  compact_threshold_ratio: 0.80
  keep_recent_turns: 6
  git_snapshot_max_chars: 16000

agent:
  max_turns: 25

lsp:
  enabled: true
  warmup_on_start: ["go"]
```

### 6.2 项目级（`.ds-code/config.yaml`）

```yaml
llm:
  model: deepseek-v4-flash
permission:
  mode: readonly
context:
  tool_result_max_chars: 50000
```

### 6.3 命令行覆盖

```bash
export DS_CODE_DEEPSEEK_API_KEY=sk-...   # 或 export DEEPSEEK_API_KEY=sk-...
ds-code --model deepseek-v4-flash --permission-mode readonly -p "解释 main 函数"
```

---

## 7. 安全与禁忌

| 禁止 | 原因 |
|------|------|
| 用户级 / 项目级 YAML 写 `llm.api_key` | 仅允许环境变量；易误入 git |
| YAML 写 `session` / `audit.log_path` / `checkpoint.*` 等路径键 | 项目目录内路径均固定 §2.1 |
| 环境变量 / CLI 修改 `~/.ds-code` 或用户 `config.yaml` 路径 | 数据根与用户配置路径均固定 |
| CLI 传 API Key | 与 §3.1 冲突 |
| 日志打印完整 `api_key` | S1 密钥泄露 |
| `--dangerously-auto` 在无隔离 CI 中默认开启 | 任意 tool 无确认 |

---

## 8. 变更记录

| 日期 | 变更 |
|------|------|
| 2026-05-16 | 初版：从 DESIGN §12 / llm-deepseek 抽离；统一 `~/.ds-code/` 路径 |
| 2026-05-16 | `llm.api_key` 仅 env：`DS_CODE_DEEPSEEK_API_KEY` → `DEEPSEEK_API_KEY` |
| 2026-05-16 | 运行时数据按项目：`~/.ds-code/projects/<sha256(project_root)>/` |
| 2026-05-16 | 移除 `session.db_path` 自定义；DB 路径仅 §2.1 |
| 2026-05-16 | 用户数据根固定 `~/.ds-code/`；移除 `DS_CODE_HOME` / `DS_CODE_CONFIG` / `--config` |
| 2026-05-16 | 项目目录内 `sessions.db` / `audit.jsonl` / `checkpoints/` 路径均不可配置；`--audit-log` 仅为开关 |
| 2026-05-16 | v1.1：`agent`/`tools` 键；compact A/B/C；`git_snapshot_max_chars`；`@dir` 限制；非 TTY ask 行为 |
| 2026-05-16 | v1.2：`lsp.*` 与 `lsp.servers`；多语言 diagnostics（DESIGN §9.5） |
