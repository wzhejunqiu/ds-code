import { useCallback, useEffect, useState, type ComponentProps } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import type { ServiceStatusView } from "../../../bindings/github.com/wzhejunqiu/ds-code/desktop/workspace/models";
import { useAppState } from "@/state/app-store";
import type { SettingsSection } from "./constants";
import { SettingsNav } from "./SettingsNav";
import { AboutPanel } from "./panels/AboutPanel";
import { ApiKeyPanel } from "./panels/ApiKeyPanel";
import { AppearancePanel } from "./panels/AppearancePanel";
import { LSPPanel } from "./panels/LSPPanel";
import { MCPPanel, type MCPLSPConfig } from "./panels/MCPPanel";
import { ModelPanel } from "./panels/ModelPanel";
import { PermissionPanel } from "./panels/PermissionPanel";
import { TracingPanel } from "./panels/TracingPanel";

function SettingsPanel({ section, services }: { section: SettingsSection; services: ServicesState }) {
  switch (section) {
    case "apiKey":
      return <ApiKeyPanel />;
    case "model":
      return <ModelPanel />;
    case "permission":
      return <PermissionPanel />;
    case "appearance":
      return <AppearancePanel />;
    case "mcp":
      return <MCPPanel {...services.mcpProps} />;
    case "lsp":
      return <LSPPanel status={services.status} />;
    case "tracing":
      return <TracingPanel />;
    case "about":
      return <AboutPanel />;
    default:
      return null;
  }
}

type ServicesState = {
  status: ServiceStatusView | null;
  mcpProps: ComponentProps<typeof MCPPanel>;
};

export function SettingsView() {
  const navigate = useNavigate();
  const { activeWorkspaceId } = useAppState();
  const [section, setSection] = useState<SettingsSection>("apiKey");
  const [servicesScope, setServicesScope] = useState<"user" | "project">("user");
  const [configText, setConfigText] = useState("");
  const [status, setStatus] = useState<ServiceStatusView | null>(null);
  const [saveMsg, setSaveMsg] = useState("");

  const loadMCPLSP = useCallback(async () => {
    const cfg = await DesktopService.GetMCPLSPConfig(
      servicesScope,
      servicesScope === "project" ? activeWorkspaceId : "",
    );
    setConfigText(JSON.stringify(cfg, null, 2));
  }, [servicesScope, activeWorkspaceId]);

  useEffect(() => {
    void loadMCPLSP();
  }, [loadMCPLSP]);

  useEffect(() => {
    if (!activeWorkspaceId) {
      setStatus(null);
      return;
    }
    void DesktopService.ServiceStatus(activeWorkspaceId).then(setStatus);
  }, [activeWorkspaceId]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        navigate("/chat");
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [navigate]);

  const saveMCPLSP = async () => {
    setSaveMsg("");
    try {
      JSON.parse(configText) as MCPLSPConfig;
      await DesktopService.SaveMCPLSPConfig(
        servicesScope,
        servicesScope === "project" ? activeWorkspaceId : "",
        configText,
      );
      setSaveMsg("Saved. Reload services to apply.");
    } catch (e) {
      setSaveMsg(String(e));
    }
  };

  const reloadServices = async () => {
    if (!activeWorkspaceId) return;
    setSaveMsg("");
    try {
      await DesktopService.ReloadWorkspaceServices(activeWorkspaceId);
      const st = await DesktopService.ServiceStatus(activeWorkspaceId);
      setStatus(st);
      setSaveMsg("Services reloaded.");
    } catch (e) {
      setSaveMsg(String(e));
    }
  };

  const services: ServicesState = {
    status,
    mcpProps: {
      scope: servicesScope,
      onScope: setServicesScope,
      projectDisabled: !activeWorkspaceId,
      configText,
      onConfigText: setConfigText,
      status,
      saveMsg,
      onSave: () => void saveMCPLSP(),
      onReload: () => void reloadServices(),
      reloadDisabled: !activeWorkspaceId,
    },
  };

  return (
    <div className="settings-view">
      <header className="settings-header">
        <Button variant="secondary" size="sm" onClick={() => navigate("/chat")}>
          ← Back to chat
        </Button>
        <h2 className="text-lg font-semibold">Settings</h2>
        <span className="text-xs text-[var(--color-muted-foreground)]">ESC</span>
      </header>
      <div className="settings-body">
        <SettingsNav active={section} onSelect={setSection} />
        <div className="settings-panel">
          <SettingsPanel section={section} services={services} />
        </div>
      </div>
    </div>
  );
}
