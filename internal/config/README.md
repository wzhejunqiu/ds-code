# config

ds-code 运行时配置的加载与校验。

**面向用户的完整参考**（YAML 键、CLI、环境变量）：[`docs/v0.1.0/CONFIG.md`](../../docs/v0.1.0/CONFIG.md)  
**示例 YAML**：[`configs/example.yaml`](../../configs/example.yaml)

## 加载流水线（`load.go`）

```
setDefaults（defaults.go）
    → 合并 ~/.ds-code/config/config.yaml
    → 合并 <git-root>/.ds-code/config.yaml
    → rejectForbiddenKeys
    → viper.Unmarshal → Config
    → applyChangedFlags（CLI，仅 flag.Changed）
    → validate
    → ResolveProjectRoot + ProjectID + EnsureProjectDataDir
    → LoadAPIKey（可选，RequireAPIKey 时）
```

| 文件 | 职责 |
|------|------|
| `defaults.go` | Viper 内置默认值 |
| `load.go` | `Load`、`BindFlags`、YAML 合并 |
| `flags.go` | CLI → 结构体（`applyChangedFlags`、`ApplyCLIDerived`） |
| `validate.go` | 枚举/范围校验；禁止的 YAML 键 |
| `project.go` | 向上查找 git 根、项目 `config.yaml` 路径 |
| `paths.go` | `~/.ds-code` 目录布局、`project_id`、DB/审计/checkpoint 路径 |
| `apikey.go` | 仅从 `DS_CODE_DEEPSEEK_API_KEY` / `DEEPSEEK_API_KEY` 读取 |
| `types.go` | `Config` 及嵌套结构（`mapstructure` 标签） |

## 优先级

后读来源按**叶子键**覆盖先读（非整文件替换）：

1. 内置默认值  
2. 用户级 YAML  
3. 项目级 YAML  
4. CLI 标志（仅用户显式设置的 flag）  
5. `DS_CODE_*` 环境变量（Viper `AutomaticEnv`，点号键映射为 `_`）

**不在 YAML 中配置：** `APIKey`、`ProjectRoot`、`ProjectID`、`ProjectDataDir`、`Prompt`、`JSONOutput`、`LogVerbosity` — 在加载阶段或 CLI 派生逻辑中赋值。

## 项目标识

- `ResolveProjectRoot(startDir)` — 向上查找 `.git`（文件或目录）；否则使用 `startDir` 的绝对路径。
- `ProjectID` — `hex(SHA256(projectRoot))`；对应 `~/.ds-code/projects/<id>/`。
- 该目录下固定路径：`sessions.db`、`audit.jsonl`、`checkpoints/`、`logs/ds-code.log`（见 `paths.go`）。

## 配置段（`types.go`）

| 结构体 | 用途 |
|--------|------|
| `LLMConfig` | 模型、token、超时、thinking、strict tools |
| `ContextConfig` | 上下文窗口、压缩、截断、@ 引用 |
| `AgentConfig` | `max_turns`（单条用户消息的子轮次上限） |
| `ToolsConfig` | 各工具限制（read、grep、shell、task 等） |

`GrepToolConfig`：`head_limit`（250）、`timeout`（20s）、`binary`（bundled/system/path）、`binary_path`、`respect_gitignore`（默认 false）。

`GlobToolConfig`：`max_results`（100）、`respect_gitignore`（默认 false）、`include_hidden`（默认 true）；binary/timeout 复用 `tools.grep.*`。
| `PermissionConfig` | `readonly` / `ask` / `auto` |
| `BTWConfig` | `/btw` 临时旁路 |
| `MCPConfig` / `WebConfig` / `LSPConfig` | 可选集成 |

`RunMode`：`agent`（默认）或 `plan`（`--plan` 时只读工具）。

## 测试

`load_test.go` 覆盖合并顺序、禁止键与 API Key 环境变量。
