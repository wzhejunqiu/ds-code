# ds-code-tui-test

交互式 TUI 集成测试入口（仅 `-tags=tuitest` 构建）。Mock LLM 驱动完整 `tui → agent → deepseek.Client` 链路。

## 构建与运行

```bash
make build-tui-test
./bin/ds-code-tui-test
```

## Header 消息通知区（自动滚动演示）

启动后 **header 右侧** 会加载本目录 [`notices.go`](./notices.go) 中的多条演示通知：

- 敏感日志警告（模拟 `--allow-log-sensitive-data`）
- MCP 工具跳过摘要（多条 `tool@server`）
- 若干条 Harness 说明通知

换行后总行数超过可见窗口（8 行）时，通知区 **每约 4 秒自动向下滚动**，滚到底后从头循环；**无需** 键盘快捷键。

### 手动验证步骤

1. 终端宽度建议 ≥ 120 列（宽屏双栏布局；过窄时通知区移至 logo 下方）。
2. 运行 `./bin/ds-code-tui-test`，观察 header 右侧红色/灰色通知文案。
3. 等待数秒，确认可见窗口内的通知行随 tick 切换（底部可出现 `通知 x–y / n` 计数）。
4. 缩窄终端宽度后 resize，确认通知区换行仍正常、无 UTF-8 乱码。

自动化断言见 `internal/tuitest/notice_test.go`（`make test-tui`）。

退出 TUI 后，harness 会清理临时 `HOME`（含 `projects/<project_id>/`）与临时项目工作区，不会在真实 `~/.ds-code/projects/` 下残留数据。

## 其他 harness 功能

```
/tcase              # 场景 Picker
/tcase run stream-basic
```

详见 [docs/v0.1.0/TUI_INTEGRATION_TEST.md](../../docs/v0.1.0/TUI_INTEGRATION_TEST.md)。
