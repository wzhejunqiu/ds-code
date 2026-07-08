import { useEffect, useState } from "react";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import type { ServiceStatusView } from "../../../bindings/github.com/wzhejunqiu/ds-code/desktop/workspace/models";
import { useAppState } from "@/state/app-store";

export function WorkspaceOverview({ workspaceId }: { workspaceId: string }) {
  const { workspaces, chats, activeWorkspaceId } = useAppState();
  const ws = workspaces.find((w) => w.id === workspaceId);
  const [status, setStatus] = useState<ServiceStatusView | null>(null);

  useEffect(() => {
    if (!workspaceId) return;
    void DesktopService.ServiceStatus(workspaceId).then(setStatus).catch(() => setStatus(null));
  }, [workspaceId]);

  const mcpCount = status?.mcp?.configuredServers?.length ?? 0;
  const mcpConnected = status?.mcp?.connected ? "connected" : "idle";
  const lspEnabled = status?.lsp?.enabled ? "on" : "off";
  const lspStarted = status?.lsp?.servers?.filter((s) => s.started).length ?? 0;

  return (
    <div className="space-y-4 p-4 text-sm">
      <div>
        <h3 className="mb-1 font-medium">Workspace</h3>
        <p className="text-[var(--color-muted-foreground)]">{ws?.name ?? workspaceId}</p>
        <p className="truncate text-xs text-[var(--color-muted-foreground)]">{ws?.root}</p>
      </div>
      <div>
        <h3 className="mb-1 font-medium">Chats</h3>
        <p>{workspaceId === activeWorkspaceId ? chats.length : "—"} sessions in this workspace</p>
      </div>
      <div>
        <h3 className="mb-1 font-medium">Services</h3>
        <ul className="space-y-1 text-[var(--color-muted-foreground)]">
          <li>
            MCP: {mcpCount} server{mcpCount === 1 ? "" : "s"} · {mcpConnected}
          </li>
          <li>
            LSP: {lspEnabled} · {lspStarted} warmed up
          </li>
        </ul>
      </div>
      <p className="text-xs text-[var(--color-muted-foreground)]">
        Select a tool card or open History to inspect checkpoint rewind.
      </p>
    </div>
  );
}
