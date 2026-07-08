import { describe, expect, it } from "vitest";
import { looksLikeMarkdown, resolveRenderedFormat } from "./detect-format";
import { sanitizeAssistantHtml } from "./sanitize-html";

const XSS_VECTORS: { name: string; input: string }[] = [
  { name: "script tag", input: '<script>alert(1)</script><p>ok</p>' },
  { name: "img onerror", input: '<img src=x onerror="alert(1)">' },
  { name: "javascript href", input: '<a href="javascript:alert(1)">x</a>' },
  { name: "iframe", input: '<iframe src="https://evil.com"></iframe>' },
  { name: "style tag", input: "<style>body{display:none}</style><p>x</p>" },
  { name: "form", input: '<form action="/"><input type="text"></form>' },
  { name: "svg onload", input: '<svg onload="alert(1)"></svg>' },
  { name: "object embed", input: '<object data="x"></object>' },
  { name: "data uri script", input: '<a href="data:text/html,<script>alert(1)</script>">x</a>' },
  { name: "event handler div", input: '<div onclick="alert(1)">click</div>' },
  { name: "meta refresh", input: '<meta http-equiv="refresh" content="0;url=javascript:alert(1)">' },
];

describe("sanitizeAssistantHtml", () => {
  it.each(XSS_VECTORS)("blocks $name", ({ input }) => {
    const out = sanitizeAssistantHtml(input);
    expect(out.toLowerCase()).not.toMatch(/<script/i);
    expect(out.toLowerCase()).not.toMatch(/onerror\s*=/i);
    expect(out.toLowerCase()).not.toMatch(/onclick\s*=/i);
    expect(out.toLowerCase()).not.toMatch(/javascript:/i);
    expect(out.toLowerCase()).not.toMatch(/<iframe/i);
    expect(out.toLowerCase()).not.toMatch(/<form/i);
    expect(out.toLowerCase()).not.toMatch(/<object/i);
    expect(out.toLowerCase()).not.toMatch(/<svg/i);
  });

  it("preserves table structure", () => {
    const out = sanitizeAssistantHtml("<table><thead><tr><th>A</th></tr></thead><tbody><tr><td>1</td></tr></tbody></table>");
    expect(out).toContain("<table>");
    expect(out).toContain("<td>");
  });

  it("preserves details", () => {
    const out = sanitizeAssistantHtml("<details><summary>More</summary><p>body</p></details>");
    expect(out).toContain("<details>");
    expect(out).toContain("<summary>");
  });

  it("allows https links with rel", () => {
    const out = sanitizeAssistantHtml('<a href="https://example.com">link</a>');
    expect(out).toContain('href="https://example.com"');
    expect(out).toContain('rel="noopener noreferrer"');
  });
});

describe("detect-format", () => {
  it("detects markdown fences", () => {
    expect(looksLikeMarkdown("```go\nfmt.Println()\n```")).toBe(true);
  });

  it("falls back html mode to markdown when content looks like md", () => {
    const r = resolveRenderedFormat("html", "# Title\n\nparagraph");
    expect(r.format).toBe("markdown");
    expect(r.fallback).toBe(true);
  });
});
