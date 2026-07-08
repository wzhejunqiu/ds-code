package prompt

// DesktopHTMLOutputOverlay instructs the model to reply with sanitized HTML fragments (desktop only).
const DesktopHTMLOutputOverlay = `## 输出格式（桌面 HTML 模式）
- 你的回复正文必须是**合法、精简的 HTML 片段**（非完整 <html> 文档）。
- 仅使用以下标签：p, br, h1-h4, ul, ol, li, table, thead, tbody, tr, th, td,
  pre, code, blockquote, strong, em, a, details, summary, span, div。
- 禁止使用：script, style, iframe, object, embed, form, input, svg 内联事件。
- 链接仅使用 https: 协议；勿使用 javascript: 或 data:。
- 代码块使用 <pre><code>，勿用 Markdown 围栏。`
