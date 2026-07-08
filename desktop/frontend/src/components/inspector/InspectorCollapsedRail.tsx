import { PanelRightOpen } from "lucide-react";
import { Button } from "@/components/ui/button";

export function InspectorCollapsedRail({ onExpand }: { onExpand: () => void }) {
  return (
    <div className="inspector-rail flex h-full flex-col items-center gap-2 border-l border-[var(--color-border)] py-3">
      <Button
        variant="ghost"
        size="icon"
        className="h-8 w-8 shrink-0"
        onClick={onExpand}
        title="Expand Inspector (⌘⌥\\)"
      >
        <PanelRightOpen className="h-4 w-4" />
      </Button>
      <button
        type="button"
        className="text-[10px] tracking-wide text-[var(--color-muted-foreground)] [writing-mode:vertical-rl]"
        onClick={onExpand}
        title="Expand Inspector (⌘⌥\\)"
      >
        Inspector
      </button>
    </div>
  );
}
