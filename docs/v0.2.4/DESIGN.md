# v0.2.4 设计文档

> 版本：v0.2.4（M4 HTML 输出）
> 状态：待实现
> 总纲：[spec/DESIGN](../v0.2.0/spec/DESIGN.md)
> 更新日期：2026-07-07

## 1. 渲染路径

`content.delta` → `content_format` → Markdown（默认）或 DOMPurify + Shadow DOM（HTML）。

## 2. 安全栈

总纲 §14、§3.4：DOMPurify 白名单、Shadow DOM、CSP、仅 `wails://` assets。

## 3. 流式策略

段末一次性消毒渲染，避免半标签 XSS。

## 4. 安全 PoC

建议 `docs/v0.2.4/SECURITY-HTML-POC.md` 记录 OWASP 向量测试结果。
