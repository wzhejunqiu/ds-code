import { useCallback, useEffect, useReducer, useState } from "react";
import { Events } from "@wailsio/runtime";
import { Composer } from "@/components/chat/Composer";
import { MessageList } from "@/components/chat/MessageList";
import { ModeSwitcher } from "@/components/chat/ModeSwitcher";
import { Button } from "@/components/ui/button";
import {
  blocksFromHistory,
  initialTurnState,
  resolvePermissionBlock,
  turnReducer,
  type AgentEventEnvelope,
  type ChatBlock,
  type TurnState,
} from "@/protocol/agent-events";
import { useAppState } from "@/state/app-store";
import { useInspector } from "@/state/inspector-store";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";

type UIAction =
  | AgentEventEnvelope
  | { type: "user"; text: string }
  | { type: "perm"; id: string; choice: string }
  | { type: "toggle_tool"; id: string }
  | { type: "toggle_subagent"; id: string }
  | { type: "load"; blocks: ChatBlock[] };

function uiReducer(state: TurnState, action: UIAction): TurnState {
  if ("type" in action) {
    if (action.type === "user") {
      return {
        ...state,
        blocks: [
          ...state.blocks,
          { id: `user-${state.blocks.length}`, role: "user", text: action.text },
        ],
      };
    }
    if (action.type === "perm") {
      return resolvePermissionBlock(state, action.id, action.choice);
    }
    if (action.type === "toggle_tool") {
      return {
        ...state,
        blocks: state.blocks.map((b) =>
          b.role === "tool" && b.id === action.id ? { ...b, collapsed: !b.collapsed } : b,
        ),
      };
    }
    if (action.type === "toggle_subagent") {
      return {
        ...state,
        blocks: state.blocks.map((b) =>
          b.role === "subagent" && b.id === action.id
            ? { ...b, record: { ...b.record, collapsed: !b.record.collapsed } }
            : b,
        ),
      };
    }
    if (action.type === "load") {
      return { ...initialTurnState(), blocks: action.blocks };
    }
  }
  return turnReducer(state, action as AgentEventEnvelope);
}

export function ChatPanel({
  dropInsert,
  onDropConsumed,
  onSubagentsChange,
}: {
  dropInsert?: string;
  onDropConsumed?: () => void;
  onSubagentsChange?: (subs: import("@/protocol/agent-events").SubagentRecord[]) => void;
}) {
  const { activeWorkspaceId, activeChatId, chats, workspaces, createChat, addWorkspace, setLayout } =
    useAppState();
  const { openTool } = useInspector();
  const [turnState, dispatch] = useReducer(uiReducer, undefined, initialTurnState);
  const [status, setStatus] = useState("");

  const activeChat = chats.find((c) => c.id === activeChatId);
  const activeWs = workspaces.find((w) => w.id === activeWorkspaceId);

  const loadHistory = useCallback(async () => {
    if (!activeWorkspaceId || !activeChatId) return;
    const [messages] = await DesktopService.ResumeChat(activeWorkspaceId, activeChatId);
    dispatch({ type: "load", blocks: blocksFromHistory(messages ?? []) });
  }, [activeWorkspaceId, activeChatId]);

  useEffect(() => {
    void loadHistory();
  }, [loadHistory]);

  useEffect(() => {
    const off = Events.On("agent:event", (raw: { data: AgentEventEnvelope }) => {
      const event = raw.data;
      if (event.workspaceId && activeWorkspaceId && event.workspaceId !== activeWorkspaceId) {
        return;
      }
      dispatch(event);
    });
    return () => off();
  }, [activeWorkspaceId]);

  useEffect(() => {
    if (!activeWorkspaceId) return;
    const id = window.setInterval(async () => {
      const s = await DesktopService.TurnStatus(activeWorkspaceId);
      setStatus(s);
    }, 500);
    return () => window.clearInterval(id);
  }, [activeWorkspaceId]);

  useEffect(() => {
    onSubagentsChange?.(turnState.subagents);
  }, [turnState.subagents, onSubagentsChange]);

  const running = turnState.running || status === "running" || status === "waiting_permission";

  const handleInspectTool = (block: Extract<ChatBlock, { role: "tool" }>) => {
    openTool(block);
    setLayout({ rightCollapsed: false });
  };

  if (!activeWorkspaceId) {
    return (
      <main className="flex h-full flex-col items-center justify-center gap-4 p-8 text-center">
        <h2 className="text-lg font-medium">Welcome to ds-code</h2>
        <p className="max-w-md text-sm text-[var(--color-muted-foreground)]">
          Add a project workspace to start chatting with the agent.
        </p>
        <Button onClick={() => void addWorkspace()}>Open project folder</Button>
      </main>
    );
  }

  if (!activeChatId) {
    return (
      <main className="flex h-full flex-col items-center justify-center gap-4 p-8 text-center">
        <h2 className="text-lg font-medium">{activeWs?.name}</h2>
        <p className="text-sm text-[var(--color-muted-foreground)]">Create a chat to begin.</p>
        <Button onClick={() => void createChat()}>New chat</Button>
      </main>
    );
  }

  return (
    <main className="flex h-full min-h-0 flex-col">
      <header className="border-b border-[var(--color-border)] px-4 py-2 text-sm">
        <span className="font-medium">{activeWs?.name}</span>
        <span className="mx-2 text-[var(--color-muted-foreground)]">▸</span>
        <span>{activeChat?.title ?? "Chat"}</span>
        {turnState.planning && <span className="ml-3 text-xs text-blue-400">planning…</span>}
        {status === "waiting_permission" && (
          <span className="ml-3 text-xs text-amber-400">waiting approval</span>
        )}
      </header>
      <ModeSwitcher />
      <MessageList
        blocks={turnState.blocks}
        workspaceId={activeWorkspaceId}
        onPermissionResolve={(id, choice) => dispatch({ type: "perm", id, choice })}
        onToolToggle={(id) => dispatch({ type: "toggle_tool", id })}
        onToolInspect={handleInspectTool}
        onSubagentToggle={(id) => dispatch({ type: "toggle_subagent", id })}
        follow={running}
      />
      <Composer
        running={running}
        disabled={!activeWorkspaceId || !activeChatId}
        onSend={(text) => dispatch({ type: "user", text })}
        insertText={dropInsert}
        onInsertConsumed={onDropConsumed}
      />
    </main>
  );
}
