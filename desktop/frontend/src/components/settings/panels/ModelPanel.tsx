import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import type { ConfigView } from "../../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop/models";
import { useAppState } from "@/state/app-store";
import { LLM_MODELS, REASONING_EFFORTS, SETTINGS_SAVED_HINT } from "../constants";
import { ScopeToggle } from "../ScopeToggle";

export function ModelPanel() {
  const { activeWorkspaceId, getConfig, saveSettings } = useAppState();
  const [scope, setScope] = useState<"user" | "project">("user");
  const [cfg, setCfg] = useState<ConfigView | null>(null);
  const [model, setModel] = useState("");
  const [effort, setEffort] = useState("");
  const [msg, setMsg] = useState("");

  useEffect(() => {
    void (async () => {
      const c = await getConfig(scope);
      setCfg(c);
      setModel(c.model || LLM_MODELS[0]);
      setEffort(c.reasoningEffort || REASONING_EFFORTS[0]);
    })();
  }, [scope, getConfig, activeWorkspaceId]);

  const save = async () => {
    setMsg("");
    try {
      await saveSettings(scope, { model, reasoningEffort: effort });
      setMsg(SETTINGS_SAVED_HINT);
    } catch (e) {
      setMsg(String(e));
    }
  };

  return (
    <section>
      <h3 className="mb-2 text-sm font-medium">Model</h3>
      <ScopeToggle scope={scope} onScope={setScope} projectDisabled={!activeWorkspaceId} />
      {cfg && (
        <p className="mb-3 text-xs text-[var(--color-muted-foreground)]">
          Current: {cfg.model} · reasoning {cfg.reasoningEffort}
        </p>
      )}
      <div className="mb-3 space-y-3">
        <div>
          <label className="mb-1 block text-xs text-[var(--color-muted-foreground)]">Model</label>
          <select
            className="w-full max-w-md rounded border border-[var(--color-border)] bg-[var(--color-muted)] px-2 py-1.5 text-sm"
            value={model}
            onChange={(e) => setModel(e.target.value)}
          >
            {LLM_MODELS.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="mb-1 block text-xs text-[var(--color-muted-foreground)]">Reasoning effort</label>
          <div className="flex flex-wrap gap-2">
            {REASONING_EFFORTS.map((e) => (
              <Button key={e} variant={effort === e ? "default" : "secondary"} size="sm" onClick={() => setEffort(e)}>
                {e}
              </Button>
            ))}
          </div>
        </div>
      </div>
      <Button size="sm" onClick={() => void save()}>
        Save
      </Button>
      {msg && <p className="mt-2 text-xs text-[var(--color-muted-foreground)]">{msg}</p>}
    </section>
  );
}
