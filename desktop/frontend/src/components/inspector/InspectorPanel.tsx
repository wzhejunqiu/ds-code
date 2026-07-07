import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import { CheckpointTimeline } from "@/components/inspector/CheckpointTimeline";
import { DiffView } from "@/components/inspector/DiffView";
import { ToolDetail } from "@/components/inspector/ToolDetail";
import { WorkspaceOverview } from "@/components/inspector/WorkspaceOverview";
import { useInspector, type InspectorTab } from "@/state/inspector-store";
import { useAppState } from "@/state/app-store";
import type { SubagentRecord } from "@/protocol/agent-events";

const tabs: { id: InspectorTab; label: string }[] = [
  { id: "detail", label: "Detail" },
  { id: "diff", label: "Diff" },
  { id: "subagent", label: "Subagents" },
  { id: "history", label: "History" },
];

export function InspectorPanel({
  subagents,
  onSelectSubagent,
  onRewound,
  focusSubagentId,
  onFocusSubagentConsumed,
}: {
  subagents: SubagentRecord[];
  onSelectSubagent?: (id: string) => void;
  onRewound?: () => void;
  focusSubagentId?: string | null;
  onFocusSubagentConsumed?: () => void;
}) {
  const { layout, setLayout, activeWorkspaceId, activeChatId } = useAppState();
  const { tab, setTab, selection, setDiffInline, diffInline, setSelection } = useInspector();

  useEffect(() => {
    if (!focusSubagentId) return;
    setTab("subagent");
    setSelection({ kind: "subagent", id: focusSubagentId });
    onSelectSubagent?.(focusSubagentId);
    onFocusSubagentConsumed?.();
  }, [focusSubagentId, setTab, setSelection, onSelectSubagent, onFocusSubagentConsumed]);

  if (layout.rightCollapsed) {
    return (
      <div className="flex h-full items-start justify-center border-l border-[var(--color-border)] p-2">
        <Button variant="secondary" size="sm" onClick={() => setLayout({ rightCollapsed: false })}>
          Inspector
        </Button>
      </div>
    );
  }

  const bgCount = subagents.filter((s) => s.background).length;

  return (
    <aside className="inspector flex h-full min-h-0 flex-col border-l border-[var(--color-border)] bg-[var(--color-card)]">
      <div className="flex items-center justify-between border-b border-[var(--color-border)] px-2 py-2">
        <div className="flex gap-1">
          {tabs.map((t) => (
            <button
              key={t.id}
              type="button"
              className={`rounded px-2 py-1 text-xs ${tab === t.id ? "bg-[var(--color-primary)] text-white" : "hover:bg-[var(--color-muted)]"}`}
              onClick={() => setTab(t.id)}
            >
              {t.label}
              {t.id === "subagent" && bgCount > 0 ? ` (${bgCount})` : ""}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-1">
          {tab === "diff" && selection.kind === "diff" && (
            <Button variant="ghost" size="sm" onClick={() => setDiffInline(!diffInline)}>
              {diffInline ? "Inline" : "Side"}
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={() => setLayout({ rightCollapsed: true })}>
            Close
          </Button>
        </div>
      </div>
      <div className="min-h-0 flex-1">
        {tab === "detail" && selection.kind === "tool" && <ToolDetail block={selection.block} />}
        {tab === "detail" && selection.kind !== "tool" && activeWorkspaceId && (
          <WorkspaceOverview workspaceId={activeWorkspaceId} />
        )}
        {tab === "detail" && selection.kind !== "tool" && !activeWorkspaceId && (
          <div className="p-4 text-sm text-[var(--color-muted-foreground)]">
            Select a workspace to see overview.
          </div>
        )}
        {tab === "diff" && activeWorkspaceId && <DiffView workspaceId={activeWorkspaceId} />}
        {tab === "history" && activeWorkspaceId && activeChatId && (
          <CheckpointTimeline
            workspaceId={activeWorkspaceId}
            sessionId={activeChatId}
            onRewound={onRewound}
          />
        )}
        {tab === "history" && (!activeWorkspaceId || !activeChatId) && (
          <div className="p-4 text-sm text-[var(--color-muted-foreground)]">Select a chat to view checkpoint history.</div>
        )}
        {tab === "subagent" && (
          <div className="overflow-auto p-2 text-sm">
            {subagents.length === 0 && (
              <p className="text-[var(--color-muted-foreground)]">No subagents in this turn.</p>
            )}
            {subagents
              .filter((s) => s.background)
              .map((s) => (
                <button
                  key={s.id}
                  type="button"
                  className="mb-2 block w-full rounded border border-[var(--color-border)] px-3 py-2 text-left hover:bg-[var(--color-muted)]"
                  onClick={() => {
                    setSelection({ kind: "subagent", id: s.id });
                    onSelectSubagent?.(s.id);
                  }}
                >
                  <div className="font-medium">{s.label || s.agentType || s.id.slice(0, 8)}</div>
                  <div className="text-xs text-[var(--color-muted-foreground)]">{s.status}</div>
                </button>
              ))}
          </div>
        )}
      </div>
    </aside>
  );
}
