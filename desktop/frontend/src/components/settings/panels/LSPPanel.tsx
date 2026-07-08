import type { ServiceStatusView } from "../../../../bindings/github.com/wzhejunqiu/ds-code/desktop/workspace/models";

export function LSPPanel({ status }: { status: ServiceStatusView | null }) {
  return (
    <section>
      <h3 className="mb-2 text-sm font-medium">LSP</h3>
      {status ? (
        <div className="space-y-2 text-sm text-[var(--color-muted-foreground)]">
          <p>LSP: {status.lsp?.enabled ? "enabled" : "disabled"}</p>
          {status.lsp?.servers?.map((s) => (
            <p key={s.id}>
              · {s.id}: {s.started ? "started" : s.hint || "idle"}
            </p>
          ))}
        </div>
      ) : (
        <p className="text-sm text-[var(--color-muted-foreground)]">Open a workspace to view LSP status.</p>
      )}
      <p className="mt-4 text-xs text-[var(--color-muted-foreground)]">
        LSP server configuration is edited in the MCP section (shared JSON config). Reload services after saving.
      </p>
    </section>
  );
}
