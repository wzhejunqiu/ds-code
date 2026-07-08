# HTML 输出安全 PoC（phase4 / M4）

> 更新日期：2026-07-08  
> 状态：已通过（Vitest 自动化）

## 目的

验证桌面 HTML 助手输出模式在 DOMPurify 白名单 + Shadow DOM 渲染下，OWASP XSS Cheat Sheet 经典向量无法执行 JavaScript。

## 运行

```bash
cd desktop/frontend
bun run test src/render/sanitize-html.test.ts
```

## 测试向量

| # | 向量 | 预期 |
|---|------|------|
| 1 | `<script>alert(1)</script>` | script 标签剥离 |
| 2 | `<img onerror=...>` | 事件属性剥离 |
| 3 | `javascript:` href | href 移除或无效 |
| 4 | `<iframe>` | 剥离 |
| 5 | `<style>` | 剥离 |
| 6 | `<form>` / `<input>` | 剥离 |
| 7 | `<svg onload>` | 剥离 |
| 8 | `<object>` | 剥离 |
| 9 | `data:` 危险 URI | 剥离 |
| 10 | `onclick` on div | 事件属性剥离 |
| 11 | `<meta refresh>` | 剥离 |

## 合法 HTML 保留

- `<table>` / `<details>` / `<summary>` 结构保留
- `https:` 链接保留，并附加 `target=_blank` + `rel=noopener noreferrer`

## 流式策略

- 流式阶段：`pre-wrap` 显示原始 HTML 源码
- `assistant.segment_end` 后：一次性 DOMPurify 消毒 + Shadow DOM 注入

## Sign-off

- [x] PRE-1：Vitest OWASP 向量全绿
- [x] PRE-2：本文档记录测试命令与向量表
