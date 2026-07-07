import { Button } from "@/components/ui/button";
import { useAppState } from "@/state/app-store";

export function InspectorPlaceholder() {
  const { layout, setLayout } = useAppState();

  if (layout.rightCollapsed) {
    return (
      <div className="flex h-full items-start justify-center border-l border-[var(--color-border)] p-2">
        <Button variant="secondary" size="sm" onClick={() => setLayout({ rightCollapsed: false })}>
          Inspector
        </Button>
      </div>
    );
  }

  return (
    <aside className="inspector flex h-full flex-col border-l border-[var(--color-border)] bg-[var(--color-card)]">
      <div className="flex items-center justify-between border-b border-[var(--color-border)] px-3 py-2">
        <span className="text-xs font-semibold">INSPECTOR</span>
        <Button variant="ghost" size="sm" onClick={() => setLayout({ rightCollapsed: true })}>
          Close
        </Button>
      </div>
      <div className="flex flex-1 items-center justify-center p-4 text-center text-sm text-[var(--color-muted-foreground)]">
        Diff and file preview arrive in v0.2.2. Tool details can be expanded in chat cards for now.
      </div>
    </aside>
  );
}
