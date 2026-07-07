import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { ChatBlock } from "@/protocol/agent-events";

export type InspectorTab = "detail" | "diff" | "subagent" | "history";

export type InspectorSelection =
  | { kind: "none" }
  | { kind: "tool"; block: Extract<ChatBlock, { role: "tool" }> }
  | {
      kind: "diff";
      patchText: string;
      fileIndex?: number;
    }
  | { kind: "file"; path: string; offset?: number; limit?: number }
  | { kind: "subagent"; id: string };

interface InspectorState {
  tab: InspectorTab;
  selection: InspectorSelection;
  diffInline: boolean;
  setTab: (tab: InspectorTab) => void;
  setSelection: (sel: InspectorSelection) => void;
  setDiffInline: (inline: boolean) => void;
  openTool: (block: Extract<ChatBlock, { role: "tool" }>) => void;
}

const InspectorContext = createContext<InspectorState | null>(null);

export function InspectorProvider({ children }: { children: React.ReactNode }) {
  const [tab, setTab] = useState<InspectorTab>("detail");
  const [selection, setSelection] = useState<InspectorSelection>({ kind: "none" });
  const [diffInline, setDiffInline] = useState(false);

  const openTool = useCallback((block: Extract<ChatBlock, { role: "tool" }>) => {
    const name = block.name.toLowerCase();
    if (name === "apply_patch" || name === "applypatch") {
      setSelection({ kind: "diff", patchText: block.args || block.result || "" });
      setTab("diff");
      return;
    }
    if (name === "read" || name === "read_file") {
      try {
        const parsed = JSON.parse(block.args) as { path?: string; file_path?: string; offset?: number; limit?: number };
        const path = parsed.path ?? parsed.file_path ?? "";
        if (path) {
          setSelection({ kind: "file", path, offset: parsed.offset, limit: parsed.limit });
          setTab("diff");
          return;
        }
      } catch {
        /* fall through */
      }
    }
    setSelection({ kind: "tool", block });
    setTab("detail");
  }, []);

  const value = useMemo(
    () => ({ tab, selection, diffInline, setTab, setSelection, setDiffInline, openTool }),
    [tab, selection, diffInline, openTool],
  );

  return <InspectorContext.Provider value={value}>{children}</InspectorContext.Provider>;
}

export function useInspector() {
  const ctx = useContext(InspectorContext);
  if (!ctx) throw new Error("useInspector outside provider");
  return ctx;
}
