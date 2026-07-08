import { marked } from "marked";
import type { ContentFormat } from "@/render/detect-format";
import { resolveRenderedFormat } from "@/render/detect-format";
import { SanitizedHtmlBlock } from "@/render/SanitizedHtmlBlock";

marked.setOptions({ breaks: true, gfm: true });

function renderMarkdown(raw: string): string {
  try {
    return marked.parse(raw, { async: false }) as string;
  } catch {
    return raw;
  }
}

export function AssistantContent({
  raw,
  streaming,
  contentFormat,
}: {
  raw: string;
  streaming: boolean;
  contentFormat?: ContentFormat;
}) {
  if (streaming) {
    return <pre className="whitespace-pre-wrap text-sm">{raw}</pre>;
  }

  const { format, fallback } = resolveRenderedFormat(contentFormat, raw);

  return (
    <>
      {fallback && (
        <p className="mb-1 text-xs text-amber-400">已按 Markdown 渲染（模型输出非 HTML）</p>
      )}
      {format === "html" ? (
        <SanitizedHtmlBlock html={raw} />
      ) : (
        <div
          className="msg-markdown prose prose-invert max-w-none text-sm"
          dangerouslySetInnerHTML={{ __html: renderMarkdown(raw) }}
        />
      )}
    </>
  );
}
