import { useAppState } from "@/state/app-store";

export function StatusBar() {
  const { permissionMode, activeWorkspaceId } = useAppState();
  const wsStatus = activeWorkspaceId ? "ready" : "no workspace";

  return (
    <footer className="status-bar">
      <span>deepseek-v4</span>
      <span>{permissionMode}</span>
      <span>{wsStatus}</span>
    </footer>
  );
}
