import { ChevronDown, ChevronRight } from "lucide-react";
import type { SubagentRecord } from "@/protocol/agent-events";

export function SubagentCard({
  record,
  onToggle,
}: {
  record: SubagentRecord;
  onToggle: () => void;
}) {
  const status =
    record.status === "running" ? "running" : record.status === "error" ? "failed" : "done";

  return (
    <div className="tool-card mb-3 border border-[var(--color-border)]">
      <button
        type="button"
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm"
        onClick={onToggle}
      >
        {record.collapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
        <span>🧩 {record.agentType || "subagent"}</span>
        <span className="truncate text-[var(--color-muted-foreground)]">{record.label || record.prompt.slice(0, 48)}</span>
        <span className="ml-auto text-xs text-[var(--color-muted-foreground)]">{status}</span>
      </button>
      {!record.collapsed && (
        <div className="border-t border-[var(--color-border)] px-2 py-2">
          {record.tools.map((t) => (
            <div key={t.id} className="mb-1 flex gap-2 text-xs text-[var(--color-muted-foreground)]">
              <span>{t.running ? "…" : "✓"}</span>
              <span className="font-medium">{t.name}</span>
              <span className="truncate">{t.args.slice(0, 60)}</span>
            </div>
          ))}
          {record.summary && (
            <pre className="mt-2 whitespace-pre-wrap text-xs">{record.summary}</pre>
          )}
          {record.error && <pre className="mt-2 text-xs text-red-400">{record.error}</pre>}
        </div>
      )}
    </div>
  );
}
