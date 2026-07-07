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
}

export interface TurnDonePayload {
  error?: string;
  cancelled?: boolean;
}

export type ChatBlock =
  | { id: string; role: "user"; text: string }
  | { id: string; role: "assistant"; raw: string; streaming: boolean; markdown?: string }
  | { id: string; role: "tool"; name: string; args: string; result?: string; running: boolean; isError?: boolean }
  | { id: string; role: "permission"; request: PermissionRequestPayload; resolved?: boolean; allowed?: boolean }
  | { id: string; role: "system"; text: string };

export interface TurnState {
  turnId: string | null;
  running: boolean;
  blocks: ChatBlock[];
  waitingPermission: boolean;
}

export const initialTurnState = (): TurnState => ({
  turnId: null,
  running: false,
  blocks: [],
  waitingPermission: false,
});

let blockCounter = 0;
function nextBlockId(prefix: string) {
  blockCounter += 1;
  return `${prefix}-${blockCounter}`;
}

export function turnReducer(state: TurnState, event: AgentEventEnvelope): TurnState {
  switch (event.kind) {
    case "turn.started": {
      return {
        ...state,
        turnId: event.turnId,
        running: true,
        waitingPermission: false,
      };
    }
    case "content.delta": {
      const payload = event.payload as ContentDeltaPayload;
      const blocks = [...state.blocks];
      const last = blocks[blocks.length - 1];
      if (last && last.role === "assistant" && last.streaming) {
        blocks[blocks.length - 1] = { ...last, raw: last.raw + payload.delta };
      } else {
        blocks.push({
          id: nextBlockId("assistant"),
          role: "assistant",
          raw: payload.delta,
          streaming: true,
        });
      }
      return { ...state, blocks };
    }
    case "assistant.segment_end": {
      const blocks = state.blocks.map((b) =>
        b.role === "assistant" && b.streaming ? { ...b, streaming: false } : b,
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
            running: true,
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
    case "turn.done": {
      const payload = (event.payload ?? {}) as TurnDonePayload;
      const blocks = [...state.blocks];
      if (payload.cancelled) {
        blocks.push({
          id: nextBlockId("sys"),
          role: "system",
          text: "Turn stopped.",
        });
      } else if (payload.error && !payload.cancelled) {
        blocks.push({
          id: nextBlockId("sys"),
          role: "system",
          text: payload.error,
        });
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
  allowed: boolean,
): TurnState {
  return {
    ...state,
    waitingPermission: false,
    blocks: state.blocks.map((b) =>
      b.role === "permission" && b.request.id === requestId
        ? { ...b, resolved: true, allowed }
        : b,
    ),
  };
}
