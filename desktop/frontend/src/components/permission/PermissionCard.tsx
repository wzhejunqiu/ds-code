import { Button } from "@/components/ui/button";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import type { ChatBlock } from "@/protocol/agent-events";

export function PermissionCard({
  block,
  workspaceId,
  onResolve,
}: {
  block: Extract<ChatBlock, { role: "permission" }>;
  workspaceId: string;
  onResolve: (requestId: string, choice: string) => void;
}) {
  const { request, resolved, choice } = block;
  if (resolved) {
    return (
      <div className="permission-card opacity-70">
        Permission {choice} for {request.kind === "web_fetch" ? request.host : request.tool}
      </div>
    );
  }

  const isWebFetch = request.kind === "web_fetch";

  return (
    <div className="permission-card">
      <div className="mb-2 text-sm font-medium">
        {isWebFetch ? "web_fetch" : request.tool} permission
      </div>
      {request.summary && <pre className="mb-2 whitespace-pre-wrap text-xs">{request.summary}</pre>}
      {request.host && <div className="mb-2 text-xs">Host: {request.host}</div>}
      {request.url && <div className="mb-2 truncate text-xs">{request.url}</div>}
      <div className="flex flex-wrap gap-2">
        {isWebFetch ? (
          <>
            <Button
              size="sm"
              onClick={() => {
                void DesktopService.ResolvePermission(workspaceId, request.id, "allow_once");
                onResolve(request.id, "allow_once");
              }}
            >
              Allow once
            </Button>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => {
                void DesktopService.ResolvePermission(workspaceId, request.id, "allow_always");
                onResolve(request.id, "allow_always");
              }}
            >
              Always allow
            </Button>
            <Button
              size="sm"
              variant="destructive"
              onClick={() => {
                void DesktopService.ResolvePermission(workspaceId, request.id, "deny");
                onResolve(request.id, "deny");
              }}
            >
              Deny
            </Button>
          </>
        ) : (
          <>
            <Button
              size="sm"
              onClick={() => {
                void DesktopService.ResolvePermission(workspaceId, request.id, "allow");
                onResolve(request.id, "allow");
              }}
            >
              Allow
            </Button>
            <Button
              size="sm"
              variant="destructive"
              onClick={() => {
                void DesktopService.ResolvePermission(workspaceId, request.id, "deny");
                onResolve(request.id, "deny");
              }}
            >
              Deny
            </Button>
          </>
        )}
      </div>
    </div>
  );
}
