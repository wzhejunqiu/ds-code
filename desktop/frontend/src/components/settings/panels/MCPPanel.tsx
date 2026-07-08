import { Button } from "@/components/ui/button";
import type { ServiceStatusView } from "../../../../bindings/github.com/wzhejunqiu/ds-code/desktop/workspace/models";
import { ScopeToggle } from "../ScopeToggle";

export type MCPLSPConfig = {
  mcp?: { servers?: { name: string; command: string; args?: string[] }[] };
  lsp?: { enabled?: boolean; servers?: Record<string, { command?: string; disabled?: boolean }> };
};

export function MCPPanel({
  scope,
  onScope,
  projectDisabled,
  configText,
  onConfigText,
  status,
  saveMsg,
  onSave,
  onReload,
  reloadDisabled,
}: {
  scope: "user" | "project";
  onScope: (scope: "user" | "project") => void;
  projectDisabled?: boolean;
  configText: string;
  onConfigText: (text: string) => void;
  status: ServiceStatusView | null;
  saveMsg: string;
  onSave: () => void;
  onReload: () => void;
  reloadDisabled?: boolean;
}) {
  return (
    <section>
      <h3 className="mb-2 text-sm font-medium">MCP</h3>
      <ScopeToggle scope={scope} onScope={onScope} projectDisabled={projectDisabled} />
      {status && (
        <div className="mb-3 space-y-2 text-xs text-[var(--color-muted-foreground)]">
          <p>MCP: {status.mcp?.connected ? "connected" : "not connected"}</p>
          {status.mcp?.configuredServers?.map((s) => (
            <p key={s.name}>
              · {s.name} ({s.command}) — {s.connected ? "ok" : "down"}
            </p>
          ))}
          {status.mcp?.skippedTools?.map((sk) => (
            <p key={`${sk.server}-${sk.tool}`} className="text-amber-400">
              skipped {sk.server}/{sk.tool}: {sk.reason}
            </p>
          ))}
        </div>
      )}
      <p className="mb-2 text-xs text-[var(--color-muted-foreground)]">
        Edit MCP/LSP config as JSON (both sections share one file).
      </p>
      <textarea
        className="mb-2 h-48 w-full rounded border border-[var(--color-border)] bg-[var(--color-muted)] p-2 font-mono text-xs"
        value={configText}
        onChange={(e) => onConfigText(e.target.value)}
      />
      <div className="flex gap-2">
        <Button size="sm" onClick={onSave}>
          Save config
        </Button>
        <Button variant="secondary" size="sm" onClick={onReload} disabled={reloadDisabled}>
          Reload services
        </Button>
      </div>
      {saveMsg && <p className="mt-2 text-xs text-[var(--color-muted-foreground)]">{saveMsg}</p>}
    </section>
  );
}
