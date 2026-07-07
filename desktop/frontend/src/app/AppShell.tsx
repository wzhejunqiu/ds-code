import { HashRouter, Navigate, Route, Routes } from "react-router-dom";
import { ChatPanel } from "@/components/chat/ChatPanel";
import { StatusBar } from "@/components/chat/StatusBar";
import { InspectorPlaceholder } from "@/components/inspector/InspectorPlaceholder";
import { Onboarding } from "@/components/settings/Onboarding";
import { SettingsView } from "@/components/settings/SettingsView";
import { WorkspaceSidebar } from "@/components/sidebar/WorkspaceSidebar";
import { useAppState } from "@/state/app-store";

function ChatLayout() {
  const { layout } = useAppState();
  const gridCols = [
    layout.leftCollapsed ? "0px" : `${layout.leftWidth}px`,
    "1fr",
    layout.rightCollapsed ? "48px" : `${layout.rightWidth}px`,
  ].join(" ");

  return (
    <div className="app-shell">
      <div className="three-col" style={{ gridTemplateColumns: gridCols }}>
        {!layout.leftCollapsed && <WorkspaceSidebar />}
        <div className="flex min-h-0 flex-col">
          <Onboarding />
          <ChatPanel />
        </div>
        <InspectorPlaceholder />
      </div>
      <StatusBar />
    </div>
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
