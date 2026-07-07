import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import { useAppState } from "@/state/app-store";

type MCPLSPConfig = {
  mcp?: { servers?: { name: string; command: string; args?: string[] }[] };
  lsp?: { enabled?: boolean; servers?: Record<string, { command?: string; disabled?: boolean }> };
};

type ServiceStatus = {
  mcp?: {
    connected?: boolean;
    configuredServers?: { name: string; command: string; connected?: boolean }[];
    skippedTools?: { server: string; tool: string; reason: string }[];
  };
  lsp?: {
    enabled?: boolean;
    servers?: { id: string; command: string; commandFound?: boolean; started?: boolean; hint?: string }[];
  };
};

export function SettingsView() {
  const { apiKeyOk, apiKeyHint, permissionMode, savePermissionMode, activeWorkspaceId } = useAppState();
  const [scope, setScope] = useState<"user" | "project">("user");
  const [configText, setConfigText] = useState("");
  const [status, setStatus] = useState<ServiceStatus | null>(null);
  const [deps, setDeps] = useState<{ name: string; ok: boolean; hint?: string }[]>([]);
  const [saveMsg, setSaveMsg] = useState("");

  const loadConfig = useCallback(async () => {
    const cfg = await DesktopService.GetMCPLSPConfig(scope, scope === "project" ? activeWorkspaceId : "");
    setConfigText(JSON.stringify(cfg, null, 2));
  }, [scope, activeWorkspaceId]);

  useEffect(() => {
    void loadConfig();
  }, [loadConfig]);

  useEffect(() => {
    if (!activeWorkspaceId) return;
    void DesktopService.ServiceStatus(activeWorkspaceId).then(setStatus);
    void DesktopService.CheckDependencies().then((rows) =>
      setDeps(rows?.map((r) => ({ name: r.name, ok: r.found, hint: r.hint })) ?? []),
    );
  }, [activeWorkspaceId]);

  const saveMCPLSP = async () => {
    setSaveMsg("");
    try {
      JSON.parse(configText) as MCPLSPConfig;
      await DesktopService.SaveMCPLSPConfig(scope, scope === "project" ? activeWorkspaceId : "", configText);
      setSaveMsg("Saved. Reload services to apply.");
    } catch (e) {
      setSaveMsg(String(e));
    }
  };

  const reloadServices = async () => {
    if (!activeWorkspaceId) return;
    setSaveMsg("");
    try {
      await DesktopService.ReloadWorkspaceServices(activeWorkspaceId);
      const st = await DesktopService.ServiceStatus(activeWorkspaceId);
      setStatus(st);
      setSaveMsg("Services reloaded.");
    } catch (e) {
      setSaveMsg(String(e));
    }
  };

  return (
    <div className="h-full overflow-y-auto p-6">
      <h2 className="mb-4 text-lg font-semibold">Settings</h2>

      <section className="mb-6">
        <h3 className="mb-2 text-sm font-medium">API Key</h3>
        <p className="text-sm text-[var(--color-muted-foreground)]">
          {apiKeyOk
            ? "API key detected via environment variables."
            : `Not configured. Set DS_CODE_DEEPSEEK_API_KEY or DEEPSEEK_API_KEY. ${apiKeyHint}`}
        </p>
      </section>

      <section className="mb-6">
        <h3 className="mb-2 text-sm font-medium">Permission mode</h3>
        <div className="flex flex-wrap gap-2">
          {(["readonly", "ask"] as const).map((mode) => (
            <Button
              key={mode}
              variant={permissionMode === mode ? "default" : "secondary"}
              size="sm"
              onClick={() => void savePermissionMode(mode)}
            >
              {mode}
            </Button>
          ))}
        </div>
      </section>

      <section className="mb-6">
        <h3 className="mb-2 text-sm font-medium">Dependencies</h3>
        <ul className="space-y-1 text-sm text-[var(--color-muted-foreground)]">
          {deps.map((d) => (
            <li key={d.name}>
              {d.name}: {d.ok ? "ok" : d.hint || "missing"}
            </li>
          ))}
        </ul>
      </section>

      <section className="mb-6">
        <div className="mb-2 flex items-center gap-2">
          <h3 className="text-sm font-medium">MCP / LSP</h3>
          <Button variant={scope === "user" ? "default" : "secondary"} size="sm" onClick={() => setScope("user")}>
            User
          </Button>
          <Button
            variant={scope === "project" ? "default" : "secondary"}
            size="sm"
            onClick={() => setScope("project")}
            disabled={!activeWorkspaceId}
          >
            Project
          </Button>
        </div>
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
            <p>LSP: {status.lsp?.enabled ? "enabled" : "disabled"}</p>
            {status.lsp?.servers?.map((s) => (
              <p key={s.id}>
                · {s.id}: {s.started ? "started" : s.hint || "idle"}
              </p>
            ))}
          </div>
        )}
        <textarea
          className="mb-2 h-48 w-full rounded border border-[var(--color-border)] bg-[var(--color-muted)] p-2 font-mono text-xs"
          value={configText}
          onChange={(e) => setConfigText(e.target.value)}
        />
        <div className="flex gap-2">
          <Button size="sm" onClick={() => void saveMCPLSP()}>
            Save config
          </Button>
          <Button variant="secondary" size="sm" onClick={() => void reloadServices()} disabled={!activeWorkspaceId}>
            Reload services
          </Button>
        </div>
        {saveMsg && <p className="mt-2 text-xs text-[var(--color-muted-foreground)]">{saveMsg}</p>}
      </section>

      <section>
        <h3 className="mb-2 text-sm font-medium">About</h3>
        <p className="text-sm text-[var(--color-muted-foreground)]">ds-code desktop v0.2.0 · phase3 (M3)</p>
      </section>
    </div>
  );
}
