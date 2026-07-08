import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import { useAppState } from "@/state/app-store";

export function Onboarding() {
  const { apiKeyOk, apiKeyHint, permissionMode, savePermissionMode, addWorkspace } = useAppState();
  const [deps, setDeps] = useState<{ name: string; found: boolean; hint?: string }[]>([]);

  useEffect(() => {
    void DesktopService.CheckDependencies().then((rows) => setDeps(rows ?? []));
  }, []);

  if (apiKeyOk) return null;

  return (
    <div className="m-4 rounded-lg border border-[var(--color-border)] bg-[var(--color-card)] p-4">
      <h3 className="mb-2 font-medium">Get started</h3>
      <ol className="mb-3 list-decimal space-y-2 pl-5 text-sm text-[var(--color-muted-foreground)]">
        <li>
          Set <code>DS_CODE_DEEPSEEK_API_KEY</code> or <code>DEEPSEEK_API_KEY</code> in your environment
          {apiKeyHint ? ` (${apiKeyHint})` : ""}.
        </li>
        <li>Choose a permission mode (current: {permissionMode}).</li>
        <li>Open your first project workspace.</li>
      </ol>
      {deps.some((d) => !d.found) && (
        <div className="mb-3 rounded border border-amber-500/40 bg-amber-500/10 p-2 text-xs">
          <div className="mb-1 font-medium text-amber-200">Optional dependencies missing</div>
          {deps
            .filter((d) => !d.found)
            .map((d) => (
              <div key={d.name}>
                <code>{d.name}</code>: {d.hint}
              </div>
            ))}
        </div>
      )}
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
        <Button size="sm" onClick={() => void addWorkspace()}>
          Open project
        </Button>
      </div>
    </div>
  );
}
