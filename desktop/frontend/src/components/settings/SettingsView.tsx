import { Button } from "@/components/ui/button";
import { useAppState } from "@/state/app-store";

export function SettingsView() {
  const { apiKeyOk, apiKeyHint, permissionMode, savePermissionMode } = useAppState();

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
        <p className="mt-2 text-xs text-[var(--color-muted-foreground)]">
          Auto mode requires CLI flags; not exposed in desktop settings.
        </p>
      </section>

      <section>
        <h3 className="mb-2 text-sm font-medium">About</h3>
        <p className="text-sm text-[var(--color-muted-foreground)]">ds-code desktop v0.2.0 · phase1 (M1 MVP)</p>
      </section>
    </div>
  );
}
