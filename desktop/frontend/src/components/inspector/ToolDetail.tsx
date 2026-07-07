import type { ChatBlock } from "@/protocol/agent-events";

const CHUNK = 16000;

export function ToolDetail({ block }: { block: Extract<ChatBlock, { role: "tool" }> }) {
  const result = block.result ?? "";
  const chunks = result.length > CHUNK ? chunkText(result, CHUNK) : [result];

  return (
    <div className="flex h-full flex-col overflow-auto p-3 text-xs">
      <div className="mb-2 font-semibold">{block.name}</div>
      {block.mcpServer && (
        <div className="mb-2 text-[var(--color-muted-foreground)]">MCP: {block.mcpServer}</div>
      )}
      <div className="mb-2 text-[var(--color-muted-foreground)]">Arguments</div>
      <pre className="mb-3 max-h-40 overflow-auto whitespace-pre-wrap rounded bg-[var(--color-muted)] p-2">
        {block.args}
      </pre>
      {block.command && (
        <>
          <div className="mb-1 text-[var(--color-muted-foreground)]">Command</div>
          <pre className="mb-3 whitespace-pre-wrap rounded bg-[var(--color-muted)] p-2">{block.command}</pre>
        </>
      )}
      <div className="mb-1 text-[var(--color-muted-foreground)]">Result</div>
      {chunks.map((c, i) => (
        <pre key={i} className="mb-2 max-h-96 overflow-auto whitespace-pre-wrap rounded bg-[var(--color-muted)] p-2">
          {c}
        </pre>
      ))}
    </div>
  );
}

function chunkText(text: string, size: number): string[] {
  const out: string[] = [];
  for (let i = 0; i < text.length; i += size) {
    out.push(text.slice(i, i + size));
  }
  return out;
}
