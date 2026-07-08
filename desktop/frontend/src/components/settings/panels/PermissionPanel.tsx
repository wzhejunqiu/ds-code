import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import type { ConfigView } from "../../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop/models";
import { useAppState } from "@/state/app-store";
import { PERMISSION_MODES, SETTINGS_SAVED_HINT, type PermissionMode } from "../constants";
import { ScopeToggle } from "../ScopeToggle";

export function PermissionPanel() {
  const { activeWorkspaceId, permissionMode, getConfig, savePermissionMode } = useAppState();
  const [scope, setScope] = useState<"user" | "project">("user");
  const [localMode, setLocalMode] = useState(permissionMode);
  const [cfg, setCfg] = useState<ConfigView | null>(null);
  const [msg, setMsg] = useState("");

  useEffect(() => {
    void (async () => {
      const c = await getConfig(scope);
      setCfg(c);
      setLocalMode(c.permissionMode);
    })();
  }, [scope, getConfig, activeWorkspaceId]);

  const select = async (mode: PermissionMode) => {
    setMsg("");
    try {
      await savePermissionMode(mode, scope);
      setLocalMode(mode);
      setMsg(SETTINGS_SAVED_HINT);
    } catch (e) {
      setMsg(String(e));
    }
  };

  return (
    <section>
      <h3 className="mb-2 text-sm font-medium">Permissions</h3>
      <ScopeToggle scope={scope} onScope={setScope} projectDisabled={!activeWorkspaceId} />
      {cfg && (
        <p className="mb-3 text-xs text-[var(--color-muted-foreground)]">Current: {cfg.permissionMode}</p>
      )}
      <div className="flex flex-wrap gap-2">
        {PERMISSION_MODES.map((mode) => (
          <Button
            key={mode}
            variant={localMode === mode ? "default" : "secondary"}
            size="sm"
            onClick={() => void select(mode)}
          >
            {mode}
          </Button>
        ))}
      </div>
      <p className="mt-3 text-xs text-[var(--color-muted-foreground)]">
        readonly — no writes; ask — prompt before writes; auto — allow writes without prompting (S3 denylist still
        applies).
      </p>
      {msg && <p className="mt-2 text-xs text-[var(--color-muted-foreground)]">{msg}</p>}
    </section>
  );
}
