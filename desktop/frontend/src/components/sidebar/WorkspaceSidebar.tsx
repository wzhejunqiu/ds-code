import { FolderPlus, MessageSquarePlus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
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

  const activeWs = workspaces.find((w) => w.id === activeWorkspaceId);

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
                {chats.map((chat) => (
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
                {chats.length === 0 && (
                  <p className="px-2 text-xs text-[var(--color-muted-foreground)]">No chats yet</p>
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
