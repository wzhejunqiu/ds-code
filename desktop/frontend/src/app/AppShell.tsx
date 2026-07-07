import { useEffect, useState } from "react";
import { HashRouter, Navigate, Route, Routes } from "react-router-dom";
import { Events } from "@wailsio/runtime";
import { ChatPanel } from "@/components/chat/ChatPanel";
import { StatusBar } from "@/components/chat/StatusBar";
import { CommandPalette } from "@/components/command/CommandPalette";
import { InspectorPanel } from "@/components/inspector/InspectorPanel";
import { Onboarding } from "@/components/settings/Onboarding";
import { SettingsView } from "@/components/settings/SettingsView";
import { WorkspaceSidebar } from "@/components/sidebar/WorkspaceSidebar";
import type { SubagentRecord } from "@/protocol/agent-events";
import { useAppState } from "@/state/app-store";
import { InspectorProvider } from "@/state/inspector-store";

function ChatLayout() {
  const { layout, addWorkspace } = useAppState();
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [dropInsert, setDropInsert] = useState("");
  const [subagents, setSubagents] = useState<SubagentRecord[]>([]);
  const gridCols = [
    layout.leftCollapsed ? "0px" : `${layout.leftWidth}px`,
    "1fr",
    layout.rightCollapsed ? "48px" : `${layout.rightWidth}px`,
  ].join(" ");

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setPaletteOpen(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  useEffect(() => {
    const off = Events.On("desktop:action", (raw: { data: Record<string, string> }) => {
      if (raw.data?.action === "open_command_palette") setPaletteOpen(true);
    });
    return () => off();
  }, []);

  return (
    <InspectorProvider>
      <div
        className="app-shell"
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault();
          const items = e.dataTransfer.items;
          if (items?.[0]?.kind === "file") {
            const file = e.dataTransfer.files[0];
            const path = (file as File & { path?: string }).path ?? file.name;
            if (file.type === "" && !path.includes(".")) {
              void addWorkspace(path);
            } else {
              setDropInsert(`@${path}`);
            }
          }
        }}
      >
        <div className="three-col" style={{ gridTemplateColumns: gridCols }}>
          {!layout.leftCollapsed && <WorkspaceSidebar />}
          <div className="flex min-h-0 flex-col">
            <Onboarding />
            <ChatPanel
              dropInsert={dropInsert}
              onDropConsumed={() => setDropInsert("")}
              onSubagentsChange={setSubagents}
            />
          </div>
          <InspectorPanel subagents={subagents} />
        </div>
        <StatusBar />
        <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} />
      </div>
    </InspectorProvider>
  );
}

export function AppShell() {
  return (
    <HashRouter>
      <Routes>
        <Route path="/" element={<Navigate to="/chat" replace />} />
        <Route path="/chat" element={<ChatLayout />} />
        <Route
          path="/settings"
          element={
            <div className="app-shell">
              <div className="three-col" style={{ gridTemplateColumns: "260px 1fr" }}>
                <WorkspaceSidebar />
                <SettingsView />
              </div>
              <StatusBar />
            </div>
          }
        />
      </Routes>
    </HashRouter>
  );
}
