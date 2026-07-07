import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Events } from "@wailsio/runtime";
import { DesktopService } from "../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import type { ChatSummary, Summary } from "../../bindings/github.com/wzhejunqiu/ds-code/desktop/workspace/models";

export interface LayoutState {
  leftWidth: number;
  rightWidth: number;
  leftCollapsed: boolean;
  rightCollapsed: boolean;
}

interface AppState {
  workspaces: Summary[];
  activeWorkspaceId: string;
  chats: ChatSummary[];
  activeChatId: string;
  layout: LayoutState;
  apiKeyOk: boolean;
  apiKeyHint: string;
  permissionMode: string;
  refreshWorkspaces: () => Promise<void>;
  addWorkspace: (path?: string) => Promise<void>;
  removeWorkspace: (id: string) => Promise<void>;
  switchWorkspace: (id: string) => Promise<void>;
  refreshChats: () => Promise<void>;
  createChat: () => Promise<string>;
  selectChat: (id: string) => Promise<void>;
  setLayout: (patch: Partial<LayoutState>) => void;
  refreshApiKey: () => Promise<void>;
  refreshConfig: () => Promise<void>;
  savePermissionMode: (mode: string) => Promise<void>;
}

const AppContext = createContext<AppState | null>(null);

const defaultLayout: LayoutState = {
  leftWidth: 260,
  rightWidth: 320,
  leftCollapsed: false,
  rightCollapsed: true,
};

export function AppProvider({ children }: { children: React.ReactNode }) {
  const [workspaces, setWorkspaces] = useState<Summary[]>([]);
  const [activeWorkspaceId, setActiveWorkspaceId] = useState("");
  const [chats, setChats] = useState<ChatSummary[]>([]);
  const [activeChatId, setActiveChatId] = useState("");
  const [layout, setLayoutState] = useState<LayoutState>(defaultLayout);
  const [apiKeyOk, setApiKeyOk] = useState(false);
  const [apiKeyHint, setApiKeyHint] = useState("");
  const [permissionMode, setPermissionMode] = useState("ask");

  const refreshWorkspaces = useCallback(async () => {
    const list = await DesktopService.ListWorkspaces();
    setWorkspaces(list ?? []);
    const active = list?.find((w) => w.active)?.id ?? "";
    setActiveWorkspaceId(active);
  }, []);

  const refreshChats = useCallback(async () => {
    if (!activeWorkspaceId) {
      setChats([]);
      return;
    }
    const list = await DesktopService.ListChats(activeWorkspaceId);
    setChats(list ?? []);
  }, [activeWorkspaceId]);

  const refreshApiKey = useCallback(async () => {
    const [ok, hint] = await DesktopService.APIKeyStatus();
    setApiKeyOk(ok);
    setApiKeyHint(hint ?? "");
  }, []);

  const refreshConfig = useCallback(async () => {
    if (!activeWorkspaceId) return;
    const cfg = await DesktopService.GetConfig("user", "");
    setPermissionMode(cfg.permissionMode);
  }, [activeWorkspaceId]);

  useEffect(() => {
    void (async () => {
      await refreshWorkspaces();
      await refreshApiKey();
      const wl = await DesktopService.GetWindowLayout();
      setLayoutState({
        leftWidth: wl.leftWidth || defaultLayout.leftWidth,
        rightWidth: wl.rightWidth || defaultLayout.rightWidth,
        leftCollapsed: wl.leftCollapsed ?? defaultLayout.leftCollapsed,
        rightCollapsed: wl.rightCollapsed ?? defaultLayout.rightCollapsed,
      });
    })();
  }, [refreshWorkspaces, refreshApiKey]);

  useEffect(() => {
    void refreshChats();
    void refreshConfig();
  }, [activeWorkspaceId, refreshChats, refreshConfig]);

  useEffect(() => {
    const off = Events.On("desktop:action", (raw: { data: Record<string, string> }) => {
      const action = raw.data?.action;
      if (action === "open_settings") {
        window.location.hash = "#/settings";
      }
      if (action === "toggle_sidebar") {
        setLayoutState((l) => ({ ...l, leftCollapsed: !l.leftCollapsed }));
      }
      if (action === "toggle_inspector") {
        setLayoutState((l) => ({ ...l, rightCollapsed: !l.rightCollapsed }));
      }
      if (action === "workspace_added" || action === "chat_created") {
        void refreshWorkspaces();
        void refreshChats();
      }
    });
    return () => off();
  }, [refreshWorkspaces, refreshChats]);

  const persistLayout = useCallback(async (next: LayoutState) => {
    await DesktopService.SaveWindowLayout({
      leftWidth: next.leftWidth,
      rightWidth: next.rightWidth,
      leftCollapsed: next.leftCollapsed,
      rightCollapsed: next.rightCollapsed,
    });
  }, []);

  const setLayout = useCallback(
    (patch: Partial<LayoutState>) => {
      setLayoutState((prev) => {
        const next = { ...prev, ...patch };
        void persistLayout(next);
        return next;
      });
    },
    [persistLayout],
  );

  const addWorkspace = useCallback(
    async (path?: string) => {
      const root = path ?? (await DesktopService.PickFolder());
      if (!root) return;
      await DesktopService.AddWorkspace(root);
      await refreshWorkspaces();
    },
    [refreshWorkspaces],
  );

  const removeWorkspace = useCallback(
    async (id: string) => {
      await DesktopService.RemoveWorkspace(id);
      await refreshWorkspaces();
    },
    [refreshWorkspaces],
  );

  const switchWorkspace = useCallback(
    async (id: string) => {
      await DesktopService.SwitchWorkspace(id);
      setActiveWorkspaceId(id);
      setActiveChatId("");
      await refreshWorkspaces();
    },
    [refreshWorkspaces],
  );

  const createChat = useCallback(async () => {
    if (!activeWorkspaceId) return "";
    const chat = await DesktopService.CreateChat(activeWorkspaceId);
    await refreshChats();
    setActiveChatId(chat.id);
    return chat.id;
  }, [activeWorkspaceId, refreshChats]);

  const selectChat = useCallback(async (id: string) => {
    setActiveChatId(id);
  }, []);

  const savePermissionMode = useCallback(
    async (mode: string) => {
      await DesktopService.SaveConfigPatch("user", "", mode);
      setPermissionMode(mode);
    },
    [],
  );

  const value = useMemo<AppState>(
    () => ({
      workspaces,
      activeWorkspaceId,
      chats,
      activeChatId,
      layout,
      apiKeyOk,
      apiKeyHint,
      permissionMode,
      refreshWorkspaces,
      addWorkspace,
      removeWorkspace,
      switchWorkspace,
      refreshChats,
      createChat,
      selectChat,
      setLayout,
      refreshApiKey,
      refreshConfig,
      savePermissionMode,
    }),
    [
      workspaces,
      activeWorkspaceId,
      chats,
      activeChatId,
      layout,
      apiKeyOk,
      apiKeyHint,
      permissionMode,
      refreshWorkspaces,
      addWorkspace,
      removeWorkspace,
      switchWorkspace,
      refreshChats,
      createChat,
      selectChat,
      setLayout,
      refreshApiKey,
      refreshConfig,
      savePermissionMode,
    ],
  );

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useAppState() {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useAppState outside provider");
  return ctx;
}
