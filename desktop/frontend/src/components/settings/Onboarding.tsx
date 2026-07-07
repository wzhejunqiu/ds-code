import { Button } from "@/components/ui/button";
import { useAppState } from "@/state/app-store";

export function Onboarding() {
  const { apiKeyOk, apiKeyHint, permissionMode, savePermissionMode, addWorkspace } = useAppState();

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
