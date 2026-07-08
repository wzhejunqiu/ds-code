import { Button } from "@/components/ui/button";

export function ScopeToggle({
  scope,
  onScope,
  projectDisabled,
}: {
  scope: "user" | "project";
  onScope: (scope: "user" | "project") => void;
  projectDisabled?: boolean;
}) {
  return (
    <div className="mb-4 flex items-center gap-2">
      <span className="text-xs text-[var(--color-muted-foreground)]">Scope</span>
      <Button variant={scope === "user" ? "default" : "secondary"} size="sm" onClick={() => onScope("user")}>
        User
      </Button>
      <Button
        variant={scope === "project" ? "default" : "secondary"}
        size="sm"
        onClick={() => onScope("project")}
        disabled={projectDisabled}
      >
        Project
      </Button>
    </div>
  );
}
