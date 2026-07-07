import { useEffect, useState } from "react";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import { useAppState } from "@/state/app-store";

export function ModeSwitcher() {
  const { activeWorkspaceId, activeChatId, permissionMode, savePermissionMode } = useAppState();
  const [runMode, setRunMode] = useStateRunMode(activeWorkspaceId, activeChatId);

  if (!activeWorkspaceId || !activeChatId) return null;

  const setMode = async (mode: string) => {
    await DesktopService.SetRunMode(activeWorkspaceId, activeChatId, mode);
    setRunMode(mode);
  };

  return (
    <div className="flex flex-wrap items-center gap-2 border-b border-[var(--color-border)] px-4 py-2 text-xs">
      <span className="text-[var(--color-muted-foreground)]">Mode</span>
      {(["agent", "plan"] as const).map((m) => (
        <button
          key={m}
          type="button"
          className={`rounded px-2 py-1 capitalize ${runMode === m ? "bg-[var(--color-primary)] text-white" : "bg-[var(--color-muted)]"}`}
          onClick={() => void setMode(m)}
        >
          {m}
        </button>
      ))}
      <span className="mx-2 text-[var(--color-border)]">|</span>
      <span className="text-[var(--color-muted-foreground)]">Permissions</span>
      {(["readonly", "ask", "auto"] as const).map((m) => (
        <button
          key={m}
          type="button"
          className={`rounded px-2 py-1 ${permissionMode === m ? "bg-[var(--color-primary)] text-white" : "bg-[var(--color-muted)]"}`}
          onClick={() => void savePermissionMode(m)}
        >
          {m}
        </button>
      ))}
      {runMode === "plan" && <span className="ml-2 text-amber-400">Plan: read-only tools</span>}
    </div>
  );
}

function useStateRunMode(wsId: string, chatId: string) {
  const [runMode, setRunMode] = useState("agent");
  useEffect(() => {
    if (!wsId || !chatId) return;
    void DesktopService.SessionRunMode(wsId, chatId)
      .then(setRunMode)
      .catch(() => setRunMode("agent"));
  }, [wsId, chatId]);
  return [runMode, setRunMode] as const;
}
