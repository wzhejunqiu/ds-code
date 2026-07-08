import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import type { ConfigView } from "../../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop/models";
import { useAppState } from "@/state/app-store";
import { SETTINGS_SAVED_HINT } from "../constants";
import { ScopeToggle } from "../ScopeToggle";

export function TracingPanel() {
  const { activeWorkspaceId, getConfig, saveSettings } = useAppState();
  const [scope, setScope] = useState<"user" | "project">("user");
  const [enabled, setEnabled] = useState(false);
  const [exporter, setExporter] = useState("");
  const [otlpEndpoint, setOtlpEndpoint] = useState("");
  const [cfg, setCfg] = useState<ConfigView | null>(null);
  const [msg, setMsg] = useState("");

  useEffect(() => {
    void (async () => {
      const c = await getConfig(scope);
      setCfg(c);
      setEnabled(c.tracingEnabled);
      setExporter(c.tracingExporter || "");
      setOtlpEndpoint(c.tracingOtlpEndpoint || "");
    })();
  }, [scope, getConfig, activeWorkspaceId]);

  const save = async () => {
    setMsg("");
    try {
      await saveSettings(scope, {
        tracingEnabled: enabled,
        tracingExporter: exporter,
        tracingOtlpEndpoint: otlpEndpoint,
      });
      setMsg(SETTINGS_SAVED_HINT);
    } catch (e) {
      setMsg(String(e));
    }
  };

  return (
    <section>
      <h3 className="mb-2 text-sm font-medium">Tracing</h3>
      <ScopeToggle scope={scope} onScope={setScope} projectDisabled={!activeWorkspaceId} />
      {cfg && (
        <p className="mb-3 text-xs text-[var(--color-muted-foreground)]">
          Current: {cfg.tracingEnabled ? "enabled" : "disabled"}
          {cfg.tracingExporter ? ` · ${cfg.tracingExporter}` : ""}
        </p>
      )}
      <label className="mb-3 flex items-center gap-2 text-sm">
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
        Enable tracing
      </label>
      <div className="mb-3">
        <label className="mb-1 block text-xs text-[var(--color-muted-foreground)]">Exporter</label>
        <select
          className="w-full max-w-md rounded border border-[var(--color-border)] bg-[var(--color-muted)] px-2 py-1.5 text-sm"
          value={exporter}
          onChange={(e) => setExporter(e.target.value)}
        >
          <option value="">(none)</option>
          <option value="log">log</option>
          <option value="otlp">otlp</option>
        </select>
      </div>
      {exporter === "otlp" && (
        <div className="mb-3">
          <label className="mb-1 block text-xs text-[var(--color-muted-foreground)]">OTLP endpoint</label>
          <input
            type="text"
            className="w-full max-w-lg rounded border border-[var(--color-border)] bg-[var(--color-muted)] px-2 py-1.5 text-sm"
            value={otlpEndpoint}
            onChange={(e) => setOtlpEndpoint(e.target.value)}
            placeholder="http://localhost:4318"
          />
        </div>
      )}
      <Button size="sm" onClick={() => void save()}>
        Save
      </Button>
      {msg && <p className="mt-2 text-xs text-[var(--color-muted-foreground)]">{msg}</p>}
    </section>
  );
}
