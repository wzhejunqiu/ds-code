import { ChevronDown, ChevronRight, ExternalLink } from "lucide-react";
import type { ChatBlock } from "@/protocol/agent-events";
import { mcpServerFromToolName } from "@/protocol/agent-events";

function summarizeTool(name: string, args: string): string {
  try {
    const parsed = JSON.parse(args) as Record<string, unknown>;
    if (name === "read" || name === "Read") {
      const path = parsed.path ?? parsed.file_path ?? "";
      const offset = parsed.offset ?? "";
      const limit = parsed.limit ?? "";
      return `${String(path)} ${offset ? `L${offset}` : ""}${limit ? `–${Number(offset) + Number(limit)}` : ""}`.trim();
    }
    if (name === "grep" || name === "Grep") {
      return String(parsed.pattern ?? args.slice(0, 80));
    }
    if (name === "glob" || name === "Glob") {
      return String(parsed.glob_pattern ?? parsed.pattern ?? "");
    }
    if (name === "bash" || name === "Shell") {
      return String(parsed.command ?? "").slice(0, 120);
    }
    if (name === "apply_patch" || name === "ApplyPatch") {
      return args.slice(0, 120);
    }
  } catch {
    /* fall through */
  }
  return args.slice(0, 100);
}

export function ToolCard({
  block,
  onToggle,
  onInspect,
  selected,
}: {
  block: Extract<ChatBlock, { role: "tool" }>;
  onToggle: () => void;
  onInspect?: () => void;
  selected?: boolean;
}) {
  const summary = summarizeTool(block.name, block.args);
  const status = block.running ? "running" : block.isError ? "failed" : "done";
  const mcp = block.mcpServer ?? mcpServerFromToolName(block.name);
  const canInspect =
    block.name.toLowerCase().includes("apply_patch") ||
    block.name.toLowerCase() === "read" ||
    block.name.toLowerCase() === "read_file";

  return (
    <div className={`tool-card ${block.isError ? "error" : ""} ${selected ? "ring-1 ring-[var(--color-primary)]" : ""}`}>
      <div className="flex items-center gap-2 px-3 py-2 text-sm">
        <button type="button" className="flex flex-1 items-center gap-2 text-left" onClick={onToggle}>
          {block.collapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          <span className="font-medium">{block.name}</span>
          {mcp && (
            <span className="rounded bg-[var(--color-muted)] px-1 text-[10px] text-[var(--color-muted-foreground)]">
              {mcp}
            </span>
          )}
          <span className="truncate text-[var(--color-muted-foreground)]">{summary}</span>
          <span className="ml-auto text-xs text-[var(--color-muted-foreground)]">{status}</span>
        </button>
        {onInspect && (
          <button
            type="button"
            className="rounded p-1 hover:bg-[var(--color-muted)]"
            title="Open in Inspector"
            onClick={(e) => {
              e.stopPropagation();
              onInspect();
            }}
          >
            <ExternalLink className="h-4 w-4" />
          </button>
        )}
      </div>
      {!block.collapsed && (
        <div className="border-t border-[var(--color-border)] px-3 py-2 text-xs">
          {block.command && <pre className="mb-2 whitespace-pre-wrap">{block.command}</pre>}
          <pre className="max-h-64 overflow-auto whitespace-pre-wrap">{block.result ?? block.args}</pre>
          {canInspect && onInspect && (
            <button type="button" className="mt-2 text-[var(--color-primary)] underline" onClick={onInspect}>
              View in Inspector
            </button>
          )}
        </div>
      )}
    </div>
  );
}
