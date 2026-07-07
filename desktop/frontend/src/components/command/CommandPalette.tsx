import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { SLASH_COMMANDS } from "@/protocol/agent-events";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import { useAppState } from "@/state/app-store";

export function CommandPalette({ open, onClose }: { open: boolean; onClose: () => void }) {
  const navigate = useNavigate();
  const { workspaces, activeWorkspaceId, activeChatId, switchWorkspace, selectChat, createChat } =
    useAppState();
  const [query, setQuery] = useState("");

  useEffect(() => {
    if (!open) setQuery("");
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  const items = useMemo(() => {
    const q = query.trim().toLowerCase();
    const cmds = SLASH_COMMANDS.map((c) => ({
      id: `cmd-${c.name}`,
      label: `/${c.name}`,
      hint: c.description,
      run: async () => {
        if (!activeWorkspaceId || !activeChatId) return;
        await DesktopService.ExecuteSlash(activeWorkspaceId, activeChatId, `/${c.name}`);
      },
    }));
    const nav = [
      ...workspaces.map((w) => ({
        id: `ws-${w.id}`,
        label: `Switch workspace: ${w.name}`,
        hint: w.root,
        run: async () => switchWorkspace(w.id),
      })),
      {
        id: "settings",
        label: "Open Settings",
        hint: "⌘,",
        run: async () => navigate("/settings"),
      },
      {
        id: "new-chat",
        label: "New chat",
        hint: "⌘N",
        run: async () => {
          const id = await createChat();
          if (id) await selectChat(id);
        },
      },
    ];
    const all = [...cmds, ...nav];
    if (!q) return all;
    return all.filter((i) => i.label.toLowerCase().includes(q) || i.hint.toLowerCase().includes(q));
  }, [query, workspaces, activeWorkspaceId, activeChatId, switchWorkspace, navigate, createChat, selectChat]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/40 pt-[20vh]" onClick={onClose}>
      <div
        className="w-full max-w-lg rounded-lg border border-[var(--color-border)] bg-[var(--color-card)] shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <input
          autoFocus
          className="w-full border-b border-[var(--color-border)] bg-transparent px-4 py-3 text-sm outline-none"
          placeholder="Search commands…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && items[0]) {
              void items[0].run();
              onClose();
            }
          }}
        />
        <ul className="max-h-72 overflow-auto py-1">
          {items.map((item) => (
            <li key={item.id}>
              <button
                type="button"
                className="flex w-full flex-col px-4 py-2 text-left text-sm hover:bg-[var(--color-muted)]"
                onClick={() => {
                  void item.run();
                  onClose();
                }}
              >
                <span>{item.label}</span>
                <span className="text-xs text-[var(--color-muted-foreground)]">{item.hint}</span>
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
