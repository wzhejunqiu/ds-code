import { useCallback, useEffect, useState } from "react";
import { DiffEditor } from "@monaco-editor/react";
import { Button } from "@/components/ui/button";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import { useInspector } from "@/state/inspector-store";

type CheckpointMeta = {
  id: number;
  tool: string;
  files?: string[];
  createdAt: number;
};

export function CheckpointTimeline({
  workspaceId,
  sessionId,
  onRewound,
}: {
  workspaceId: string;
  sessionId: string;
  onRewound?: () => void;
}) {
  const { diffInline } = useInspector();
  const [checkpoints, setCheckpoints] = useState<CheckpointMeta[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [newerIds, setNewerIds] = useState<number[]>([]);
  const [diffs, setDiffs] = useState<
    { path: string; original: string; modified: string; language: string }[]
  >([]);
  const [showDiff, setShowDiff] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [fileIndex, setFileIndex] = useState(0);

  const refresh = useCallback(async () => {
    if (!workspaceId || !sessionId) return;
    setLoading(true);
    setError("");
    try {
      const list = await DesktopService.ListCheckpoints(workspaceId, sessionId);
      const sorted = [...(list ?? [])].sort((a, b) => b.id - a.id);
      setCheckpoints(sorted);
    } catch (e) {
      setError(String(e));
      setCheckpoints([]);
    } finally {
      setLoading(false);
    }
  }, [workspaceId, sessionId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const selectCheckpoint = async (id: number) => {
    setSelectedId(id);
    setConfirming(false);
    setShowDiff(false);
    setDiffs([]);
    try {
      const newer = await DesktopService.CheckpointNewerIDs(workspaceId, sessionId, id);
      setNewerIds(newer ?? []);
    } catch {
      setNewerIds([]);
    }
  };

  const loadDiff = async () => {
    if (selectedId == null) return;
    setLoading(true);
    try {
      const d = await DesktopService.PreviewCheckpointRewind(workspaceId, sessionId, selectedId);
      setDiffs(d ?? []);
      setShowDiff(true);
      setFileIndex(0);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  };

  const doRewind = async () => {
    if (selectedId == null) return;
    setLoading(true);
    setError("");
    try {
      await DesktopService.RewindCheckpoint(workspaceId, sessionId, selectedId);
      setConfirming(false);
      setSelectedId(null);
      setShowDiff(false);
      setDiffs([]);
      await refresh();
      onRewound?.();
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  };

  const selected = checkpoints.find((c) => c.id === selectedId);

  if (!sessionId) {
    return <p className="p-4 text-sm text-[var(--color-muted-foreground)]">Select a chat to view checkpoints.</p>;
  }
  if (loading && checkpoints.length === 0) {
    return <p className="p-4 text-sm text-[var(--color-muted-foreground)]">Loading checkpoints…</p>;
  }

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden text-sm">
      <div className="border-b border-[var(--color-border)] px-3 py-2 text-xs font-medium text-[var(--color-muted-foreground)]">
        REWIND timeline (newest first)
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-2">
        {checkpoints.length === 0 && (
          <p className="px-2 py-4 text-[var(--color-muted-foreground)]">No checkpoints for this session yet.</p>
        )}
        {checkpoints.map((cp) => (
          <button
            key={cp.id}
            type="button"
            className={`mb-2 block w-full rounded border px-3 py-2 text-left ${
              selectedId === cp.id
                ? "border-[var(--color-primary)] bg-[var(--color-muted)]"
                : "border-[var(--color-border)] hover:bg-[var(--color-muted)]"
            }`}
            onClick={() => void selectCheckpoint(cp.id)}
          >
            <div className="font-medium">
              #{cp.id} · {cp.tool}
            </div>
            <div className="text-xs text-[var(--color-muted-foreground)]">
              {cp.files?.length ? cp.files.join(", ") : "no files"} ·{" "}
              {new Date(cp.createdAt).toLocaleString()}
            </div>
          </button>
        ))}
      </div>

      {selected && (
        <div className="border-t border-[var(--color-border)] p-3">
          <p className="mb-2 text-xs text-[var(--color-muted-foreground)]">
            Affected: {selected.files?.join(", ") || "none"}
          </p>
          {!showDiff && (
            <Button variant="secondary" size="sm" className="mb-2" onClick={() => void loadDiff()}>
              View rewind diff
            </Button>
          )}
          {showDiff && diffs.length > 0 && (
            <div className="mb-2 h-48 min-h-0 overflow-hidden rounded border border-[var(--color-border)]">
              {diffs.length > 1 && (
                <div className="flex flex-wrap gap-1 border-b border-[var(--color-border)] p-1">
                  {diffs.map((f, i) => (
                    <button
                      key={f.path}
                      type="button"
                      className={`rounded px-2 py-0.5 text-xs ${i === fileIndex ? "bg-[var(--color-primary)] text-white" : "bg-[var(--color-muted)]"}`}
                      onClick={() => setFileIndex(i)}
                    >
                      {f.path}
                    </button>
                  ))}
                </div>
              )}
              <DiffEditor
                height="100%"
                language={diffs[fileIndex]?.language ?? "plaintext"}
                original={diffs[fileIndex]?.original ?? ""}
                modified={diffs[fileIndex]?.modified ?? ""}
                options={{
                  readOnly: true,
                  renderSideBySide: !diffInline,
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                }}
                theme="vs-dark"
              />
            </div>
          )}
          {!confirming ? (
            <Button variant="default" size="sm" onClick={() => setConfirming(true)}>
              Rewind to #{selected.id}
            </Button>
          ) : (
            <div className="rounded border border-amber-600/50 bg-amber-950/30 p-3">
              <p className="mb-2 text-xs">
                {newerIds.length > 0
                  ? `Rewind to #${selected.id} will discard checkpoints #${newerIds.join(", #")}.`
                  : `Rewind workspace to checkpoint #${selected.id}?`}
              </p>
              <div className="flex gap-2">
                <Button size="sm" onClick={() => void doRewind()} disabled={loading}>
                  Confirm rewind
                </Button>
                <Button variant="secondary" size="sm" onClick={() => setConfirming(false)}>
                  Cancel
                </Button>
              </div>
            </div>
          )}
        </div>
      )}
      {error && <p className="px-3 pb-2 text-xs text-red-400">{error}</p>}
    </div>
  );
}
