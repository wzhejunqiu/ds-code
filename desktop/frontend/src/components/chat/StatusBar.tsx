import { useEffect, useState } from "react";
import { Events } from "@wailsio/runtime";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import type { AgentEventEnvelope } from "@/protocol/agent-events";
import { useAppState } from "@/state/app-store";

type UsageState = {
  model: string;
  promptTokens: number;
  completionTokens: number;
  costLabel: string;
};

export function StatusBar() {
  const { permissionMode, activeWorkspaceId, activeChatId } = useAppState();
  const wsStatus = activeWorkspaceId ? "ready" : "no workspace";
  const [hint, setHint] = useState("");
  const [usage, setUsage] = useState<UsageState>({
    model: "deepseek-v4",
    promptTokens: 0,
    completionTokens: 0,
    costLabel: "",
  });

  useEffect(() => {
    if (!activeWorkspaceId || !activeChatId) {
      setUsage({ model: "deepseek-v4", promptTokens: 0, completionTokens: 0, costLabel: "" });
      return;
    }
    void (async () => {
      try {
        const u = await DesktopService.SessionUsage(activeWorkspaceId, activeChatId);
        setUsage({
          model: u.model || "deepseek-v4",
          promptTokens: u.promptTokens ?? 0,
          completionTokens: u.completionTokens ?? 0,
          costLabel: u.estimatedCostLabel ?? "",
        });
      } catch {
        /* ignore */
      }
    })();
  }, [activeWorkspaceId, activeChatId]);

  useEffect(() => {
    const off = Events.On("agent:event", (raw: { data: AgentEventEnvelope }) => {
      const event = raw.data;
      if (event.kind !== "usage.update") return;
      if (event.workspaceId && activeWorkspaceId && event.workspaceId !== activeWorkspaceId) return;
      const p = event.payload as { prompt_tokens?: number; completion_tokens?: number };
      setUsage((prev) => ({
        ...prev,
        promptTokens: prev.promptTokens + (p.prompt_tokens ?? 0),
        completionTokens: prev.completionTokens + (p.completion_tokens ?? 0),
      }));
    });
    return () => off();
  }, [activeWorkspaceId]);

  useEffect(() => {
    const onDone = () => {
      if (!activeWorkspaceId || !activeChatId) return;
      void DesktopService.SessionUsage(activeWorkspaceId, activeChatId).then((u) => {
        setUsage({
          model: u.model || "deepseek-v4",
          promptTokens: u.promptTokens ?? 0,
          completionTokens: u.completionTokens ?? 0,
          costLabel: u.estimatedCostLabel ?? "",
        });
      });
    };
    window.addEventListener("ds-code:turn-done", onDone);
    return () => window.removeEventListener("ds-code:turn-done", onDone);
  }, [activeWorkspaceId, activeChatId]);

  useEffect(() => {
    const off = Events.On("desktop:hint", (raw: { data: { text?: string } }) => {
      setHint(raw.data?.text ?? "");
    });
    return () => off();
  }, []);

  useEffect(() => {
    if (!hint) return;
    const id = window.setTimeout(() => setHint(""), 5000);
    return () => window.clearTimeout(id);
  }, [hint]);

  const tokens = usage.promptTokens + usage.completionTokens;

  return (
    <footer className="status-bar">
      {hint ? <span className="status-bar-hint">{hint}</span> : null}
      <span>{usage.model}</span>
      <span>{permissionMode}</span>
      <span>{tokens > 0 ? `${tokens.toLocaleString()} tok` : "0 tok"}</span>
      {usage.costLabel && <span>{usage.costLabel}</span>}
      <span>{wsStatus}</span>
    </footer>
  );
}
