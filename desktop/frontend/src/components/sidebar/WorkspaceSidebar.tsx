import { FolderPlus, MessageSquarePlus, Search, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import type { ChatSummary } from "../../../bindings/github.com/wzhejunqiu/ds-code/desktop/workspace/models";
import { useAppState } from "@/state/app-store";

export function WorkspaceSidebar() {
  const {
    workspaces,
    activeWorkspaceId,
    chats,
    activeChatId,
    addWorkspace,
    removeWorkspace,
    switchWorkspace,
    createChat,
    selectChat,
  } = useAppState();
  const [query, setQuery] = useState("");
  const [searchResults, setSearchResults] = useState<ChatSummary[] | null>(null);

  const activeWs = workspaces.find((w) => w.id === activeWorkspaceId);

  useEffect(() => {
    if (!activeWorkspaceId) {
      setSearchResults(null);
      return;
    }
    const q = query.trim();
    if (!q) {
      setSearchResults(null);
      return;
    }
    const t = window.setTimeout(() => {
      void DesktopService.SearchChats(activeWorkspaceId, q).then(setSearchResults);
    }, 300);
    return () => window.clearTimeout(t);
  }, [query, activeWorkspaceId]);

  const displayChats = useMemo(() => {
    if (searchResults) return searchResults;
    return chats;
  }, [searchResults, chats]);

  return (
    <aside className="flex h-full flex-col border-r border-[var(--color-border)] bg-[var(--color-card)]">
      <div className="flex items-center justify-between border-b border-[var(--color-border)] px-3 py-2">
        <span className="text-xs font-semibold tracking-wide text-[var(--color-muted-foreground)]">
          WORKSPACES
        </span>
        <Button variant="ghost" size="icon" onClick={() => void addWorkspace()} title="Open folder">
          <FolderPlus className="h-4 w-4" />
        </Button>
      </div>
      {activeWorkspaceId && (
        <div className="border-b border-[var(--color-border)] px-2 py-2">
          <div className="flex items-center gap-1 rounded-md border border-[var(--color-border)] px-2 py-1">
            <Search className="h-3 w-3 text-[var(--color-muted-foreground)]" />
            <input
              className="w-full bg-transparent text-xs outline-none"
              placeholder="Search chats…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
        </div>
      )}
      <div className="min-h-0 flex-1 overflow-y-auto p-2">
        {workspaces.length === 0 && (
          <p className="px-2 py-4 text-sm text-[var(--color-muted-foreground)]">
            No workspaces yet. Open a project folder to begin.
          </p>
        )}
        {workspaces.map((ws) => (
          <div key={ws.id} className="mb-2">
            <button
              type="button"
              className={`flex w-full items-center justify-between rounded-md px-2 py-1.5 text-left text-sm hover:bg-[var(--color-muted)] ${
                ws.id === activeWorkspaceId ? "bg-[var(--color-muted)] font-medium" : ""
              } ${!ws.valid ? "opacity-50" : ""}`}
              onClick={() => void switchWorkspace(ws.id)}
            >
              <span className="truncate">{ws.name}</span>
              {ws.id === activeWorkspaceId && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 shrink-0"
                  onClick={(e) => {
                    e.stopPropagation();
                    void removeWorkspace(ws.id);
                  }}
                  title="Remove workspace"
                >
                  <Trash2 className="h-3 w-3" />
                </Button>
              )}
            </button>
            {ws.id === activeWorkspaceId && (
              <div className="mt-1 space-y-1 pl-2">
                <div className="flex items-center justify-between px-1">
                  <span className="text-xs text-[var(--color-muted-foreground)]">Chats</span>
                  <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => void createChat()}>
                    <MessageSquarePlus className="h-3 w-3" />
                  </Button>
                </div>
                {displayChats.map((chat) => (
                  <button
                    key={chat.id}
                    type="button"
                    className={`block w-full truncate rounded px-2 py-1 text-left text-xs hover:bg-[var(--color-muted)] ${
                      chat.id === activeChatId ? "bg-[var(--color-muted)]" : ""
                    }`}
                    onClick={() => void selectChat(chat.id)}
                  >
                    {chat.title || "(untitled)"}
                  </button>
                ))}
                {displayChats.length === 0 && (
                  <p className="px-2 text-xs text-[var(--color-muted-foreground)]">
                    {query.trim() ? "No matches" : "No chats yet"}
                  </p>
                )}
              </div>
            )}
          </div>
        ))}
      </div>
      {activeWs && (
        <div className="border-t border-[var(--color-border)] p-2 text-xs text-[var(--color-muted-foreground)] truncate">
          {activeWs.root}
        </div>
      )}
    </aside>
  );
}
