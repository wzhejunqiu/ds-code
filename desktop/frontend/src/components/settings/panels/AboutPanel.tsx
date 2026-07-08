import { useEffect, useState } from "react";
import { DesktopService } from "../../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";

export function AboutPanel() {
  const [deps, setDeps] = useState<{ name: string; ok: boolean; hint?: string }[]>([]);

  useEffect(() => {
    void DesktopService.CheckDependencies().then((rows) =>
      setDeps(rows?.map((r) => ({ name: r.name, ok: r.found, hint: r.hint })) ?? []),
    );
  }, []);

  return (
    <section>
      <h3 className="mb-2 text-sm font-medium">About</h3>
      <p className="mb-4 text-sm text-[var(--color-muted-foreground)]">ds-code desktop · DeepSeek V4 agent</p>
      <h4 className="mb-2 text-xs font-medium uppercase tracking-wide text-[var(--color-muted-foreground)]">
        Dependencies
      </h4>
      <ul className="space-y-1 text-sm text-[var(--color-muted-foreground)]">
        {deps.map((d) => (
          <li key={d.name}>
            {d.name}: {d.ok ? "ok" : d.hint || "missing"}
          </li>
        ))}
      </ul>
    </section>
  );
}
