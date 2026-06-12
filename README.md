# ds-code

终端里的 AI 编码助手，由 DeepSeek 驱动。在项目目录中对话式阅读、修改代码，或进入只读 Plan 模式做架构分析。

## 能做什么

- **读代码**：搜索、浏览、解释项目结构与实现
- **改代码**：按你的描述编辑文件、运行命令（改前会询问确认）
- **Plan 模式**：只读分析，输出实施方案，不改动文件
- **续聊**：会话自动保存，可恢复之前的对话
- **懂项目**：读取 `AGENTS.md`、项目 Rules 与 Skills

## 准备

需要 [DeepSeek](https://platform.deepseek.com/) API Key：

```bash
export DS_CODE_DEEPSEEK_API_KEY=sk-...
# 或
export DEEPSEEK_API_KEY=sk-...
```

## 安装

从 [GitHub Releases](https://github.com/wzhejunqiu/ds-code/releases) 下载对应平台的压缩包，解压后将 `ds-code` 放到 PATH 中。

| 你的环境 | 下载文件 |
|----------|----------|
| macOS Apple Silicon | `ds-code-darwin-arm64.tar.gz` |
| macOS Intel | `ds-code-darwin-x86_64.tar.gz` |
| Linux x86_64 | `ds-code-linux-amd64.tar.gz` |
| Linux ARM64 | `ds-code-linux-arm64.tar.gz` |
| Alpine / musl Linux x86_64 | `ds-code-linux-musl-amd64.tar.gz` |
| Alpine / musl Linux ARM64 | `ds-code-linux-musl-arm64.tar.gz` |

示例（macOS Apple Silicon）：

```bash
curl -LO https://github.com/wzhejunqiu/ds-code/releases/download/v0.1.0/ds-code-darwin-arm64.tar.gz
tar xzf ds-code-darwin-arm64.tar.gz
chmod +x ds-code
sudo mv ds-code /usr/local/bin/   # 或放入任意 PATH 目录
```

## 快速开始

```bash
export DS_CODE_DEEPSEEK_API_KEY=sk-...

# 交互界面（在项目目录下运行）
ds-code

# 单次任务（适合脚本）
ds-code -p "解释 main 函数的作用"

# 只读分析，不改文件
ds-code --plan -p "梳理 internal/agent 的结构并给出重构建议"
```

## 常用用法

**引用文件**：在输入中用 `@path/to/file.go` 或 `@src/` 预加载文件/目录。

**恢复会话**：

```bash
ds-code sessions          # 列出历史会话
ds-code resume <会话ID>
```

**权限**：默认在修改文件或运行 shell 前询问（`y/N`）。脚本中可使用 `--permission-mode readonly`（只读）或 `--dangerously-auto`（自动批准，请谨慎）。

**JSON 输出**（便于 CI）：`ds-code --json -p "..."`

## 交互快捷键

| 按键 | 作用 |
|------|------|
| `Ctrl+C` | 取消当前轮次 |
| `Ctrl+T` | 工具调用日志 |
| `Ctrl+R` | 显示/隐藏推理过程 |
| `Ctrl+L` | 上下文用量（`/context`） |

常用命令：输入 `/help` 查看全部；`/compact` 压缩上下文；`/clear` 开始新会话。

## 配置

可选。用户级：`~/.ds-code/config/config.yaml`；项目级：`<git-root>/.ds-code/config.yaml`。示例见 [`configs/example.yaml`](configs/example.yaml)。

详细选项见 [CONFIG.md](docs/CONFIG.md)。

## 更多文档

- [CONFIG.md](docs/CONFIG.md) — 配置说明
- [SECURITY.md](docs/SECURITY.md) — 安全说明
- [CONTRIBUTING.md](CONTRIBUTING.md) — 从源码构建与参与开发

## 许可证

[Apache License 2.0](LICENSE)
