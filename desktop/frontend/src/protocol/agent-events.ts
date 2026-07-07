export type AgentEventKind =
  | "turn.started"
  | "content.delta"
  | "reasoning.delta"
  | "tool.start"
  | "tool.end"
  | "assistant.segment_end"
  | "planning.start"
  | "planning.end"
  | "subagent.start"
  | "subagent.end"
  | "subagent.tool.start"
  | "subagent.tool.end"
  | "usage.update"
  | "turn.done"
  | "permission.request";

export interface AgentEventEnvelope {
  v: number;
  seq: number;
  turnId: string;
  streamId: string;
  workspaceId: string;
  kind: AgentEventKind;
  ts: number;
  critical: boolean;
  payload: unknown;
}

export interface ContentDeltaPayload {
  delta: string;
}

export interface ReasoningDeltaPayload {
  delta: string;
}

export interface ToolStartPayload {
  name: string;
  args: string;
  command?: string;
}

export interface ToolEndPayload {
  name: string;
  args: string;
  command?: string;
  result: string;
  isError: boolean;
}

export interface PermissionRequestPayload {
  id: string;
  kind: string;
  tool: string;
  summary: string;
  host?: string;
  url?: string;
}

export interface UsageUpdatePayload {
  promptTokens?: number;
  completionTokens?: number;
  maxTokens?: number;
}

export interface TurnDonePayload {
  error?: string;
  cancelled?: boolean;
}

export type ChatBlock =
  | { id: string; role: "user"; text: string }
  | {
      id: string;
      role: "assistant";
      raw: string;
      reasoning?: string;
      reasoningOpen?: boolean;
      streaming: boolean;
    }
  | {
      id: string;
      role: "tool";
      name: string;
      args: string;
      command?: string;
      result?: string;
      running: boolean;
      isError?: boolean;
      collapsed?: boolean;
    }
  | {
      id: string;
      role: "permission";
      request: PermissionRequestPayload;
      resolved?: boolean;
      choice?: string;
    }
  | { id: string; role: "system"; text: string };

export interface TurnState {
  turnId: string | null;
  running: boolean;
  waitingPermission: boolean;
  blocks: ChatBlock[];
  usage: UsageUpdatePayload;
}

export const initialTurnState = (): TurnState => ({
  turnId: null,
  running: false,
  waitingPermission: false,
  blocks: [],
  usage: {},
});

let blockCounter = 0;
export function resetBlockCounter() {
  blockCounter = 0;
}
function nextBlockId(prefix: string) {
  blockCounter += 1;
  return `${prefix}-${blockCounter}`;
}

export function turnReducer(state: TurnState, event: AgentEventEnvelope): TurnState {
  switch (event.kind) {
    case "turn.started":
      return {
        ...state,
        turnId: event.turnId,
        running: true,
        waitingPermission: false,
      };
    case "content.delta": {
      const payload = event.payload as ContentDeltaPayload;
      const blocks = [...state.blocks];
      const last = blocks[blocks.length - 1];
      if (last && last.role === "assistant" && last.streaming) {
        const updated = {
          ...last,
          raw: last.raw + payload.delta,
          reasoningOpen: false,
        };
        blocks[blocks.length - 1] = updated;
      } else {
        blocks.push({
          id: nextBlockId("assistant"),
          role: "assistant",
          raw: payload.delta,
          streaming: true,
          reasoningOpen: false,
        });
      }
      return { ...state, blocks };
    }
    case "reasoning.delta": {
      const payload = event.payload as ReasoningDeltaPayload;
      const blocks = [...state.blocks];
      const last = blocks[blocks.length - 1];
      if (last && last.role === "assistant" && last.streaming) {
        blocks[blocks.length - 1] = {
          ...last,
          reasoning: (last.reasoning ?? "") + payload.delta,
          reasoningOpen: true,
        };
      } else {
        blocks.push({
          id: nextBlockId("assistant"),
          role: "assistant",
          raw: "",
          reasoning: payload.delta,
          reasoningOpen: true,
          streaming: true,
        });
      }
      return { ...state, blocks };
    }
    case "assistant.segment_end": {
      const blocks = state.blocks.map((b) =>
        b.role === "assistant" && b.streaming ? { ...b, streaming: false, reasoningOpen: false } : b,
      );
      return { ...state, blocks };
    }
    case "tool.start": {
      const payload = event.payload as ToolStartPayload;
      return {
        ...state,
        blocks: [
          ...state.blocks,
          {
            id: nextBlockId("tool"),
            role: "tool",
            name: payload.name,
            args: payload.args,
            command: payload.command,
            running: true,
            collapsed: true,
          },
        ],
      };
    }
    case "tool.end": {
      const payload = event.payload as ToolEndPayload;
      const blocks = state.blocks.map((b) =>
        b.role === "tool" && b.name === payload.name && b.running
          ? {
              ...b,
              running: false,
              result: payload.result,
              isError: payload.isError,
              collapsed: !payload.isError,
            }
          : b,
      );
      return { ...state, blocks };
    }
    case "permission.request": {
      const payload = event.payload as PermissionRequestPayload;
      return {
        ...state,
        waitingPermission: true,
        blocks: [
          ...state.blocks,
          { id: nextBlockId("perm"), role: "permission", request: payload },
        ],
      };
    }
    case "usage.update":
      return { ...state, usage: { ...state.usage, ...(event.payload as UsageUpdatePayload) } };
    case "turn.done": {
      const payload = (event.payload ?? {}) as TurnDonePayload;
      const blocks = [...state.blocks];
      if (payload.cancelled) {
        blocks.push({ id: nextBlockId("sys"), role: "system", text: "Turn stopped." });
      } else if (payload.error) {
        blocks.push({ id: nextBlockId("sys"), role: "system", text: payload.error });
      }
      return {
        ...state,
        running: false,
        waitingPermission: false,
        blocks: blocks.map((b) =>
          b.role === "assistant" && b.streaming ? { ...b, streaming: false } : b,
        ),
      };
    }
    default:
      return state;
  }
}

export function resolvePermissionBlock(
  state: TurnState,
  requestId: string,
  choice: string,
): TurnState {
  return {
    ...state,
    waitingPermission: false,
    blocks: state.blocks.map((b) =>
      b.role === "permission" && b.request.id === requestId
        ? { ...b, resolved: true, choice }
        : b,
    ),
  };
}

export interface HistoryMessage {
  id: number;
  role: string;
  content: string;
  reasoning?: string;
  toolCalls?: string;
  toolCallId?: string;
  toolName?: string;
}

export function blocksFromHistory(messages: HistoryMessage[]): ChatBlock[] {
  resetBlockCounter();
  const blocks: ChatBlock[] = [];
  for (const m of messages) {
    if (m.role === "user") {
      blocks.push({ id: nextBlockId("user"), role: "user", text: m.content });
    } else if (m.role === "assistant") {
      blocks.push({
        id: nextBlockId("assistant"),
        role: "assistant",
        raw: m.content,
        reasoning: m.reasoning,
        streaming: false,
      });
    } else if (m.role === "tool") {
      blocks.push({
        id: nextBlockId("tool"),
        role: "tool",
        name: m.toolName || "tool",
        args: m.toolCalls || m.content,
        result: m.content,
        running: false,
        collapsed: true,
      });
    }
  }
  return blocks;
}
