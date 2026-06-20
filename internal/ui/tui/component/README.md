# component

TUI 共用的 Bubble Tea UI 组件。目前主要是 `Picker`。

## Picker

`Picker` 是浮层列表的**视图 + 键盘状态**辅助（斜杠补全、`/resume` 会话列表）。不负责 Bubble Tea 的 `Update`/`View` 接线——由调用方从业务数据同步 `Items`，调用 `HandleKey`，再用 `View()` 渲染。

### 数据模型

| 字段 | 含义 |
|------|------|
| `Items` | 展示用字符串（由调用方格式化） |
| `Cursor` | 当前选中下标 |
| `Scroll` | 分页时窗口首项下标 |
| `PageSize` | 最大可见行数；`0` 表示显示全部（无滚动窗口） |
| `Header` / `Empty` | 列表上方文案 / `Len() == 0` 时文案 |

调用方需维护**并行切片**（例如 `complete []slash.Command` 与 `completePicker.Cursor` 下标对齐），因为 `Picker` 只存字符串。

### 按键契约

`HandleKey(msg, opts) (action, handled)`：

- `handled == false` — 未消费按键；例如 `PickerTabDefault` 时 Tab 仍交给文本输入框。
- `handled == true` 且 `action == PickerKeyNone` — 仅导航；调用方应重绘（`sync*Overlay`）。
- `handled == true` 且其他 action — 调用方执行业务（应用补全、关闭浮层等）。

`PickerTabBehavior` 因两种 picker 的 Tab 语义不同：

- **斜杠补全**（`PickerTabConfirm`）：Tab 确认当前高亮项（与 Enter 一致）。
- **恢复列表**（`PickerTabMoveDown`）：Tab 下移，类似常见 CLI 菜单；Enter 在 `tui.updateKey` 中按会话 ID 恢复。

### 滚动（`ensureScrollVisible`）

`PageSize > 0` 时，列表显示滑动窗口 `[Scroll, Scroll+PageSize)`。光标变化后 `ensureScrollVisible`：

1. 将 `Cursor` 限制在 `[0, Len-1]`。
2. 若 `Cursor < Scroll`，设 `Scroll = Cursor`（向上滚）。
3. 若 `Cursor >= Scroll+page`，设 `Scroll = Cursor-page+1`（向下滚）。
4. 将 `Scroll` 限制在 `[0, Len-page]`。

`MovePage` 按 `PageSize` 跳页；仅当 `PageSize > 0` 时处理 PgUp/PgDn。

### 样式

列表使用 `theme` 色板（`styles.go` 的 `styleItem` / `styleItemSelected`）。选中行前缀为 `▸`。
