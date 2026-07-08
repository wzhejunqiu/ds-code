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
  | "permission.request"
  | "system.notice";

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

export interface SubagentStartPayload {
  id: string;
  label: string;
  prompt: string;
  agentType: string;
  background: boolean;
}

export interface SubagentEndPayload {
  id: string;
  summary: string;
  error?: string;
}

export interface SubagentToolStartPayload {
  subagentId: string;
  name: string;
  args: string;
  command?: string;
}

export interface SubagentToolEndPayload {
  subagentId: string;
  name: string;
  args: string;
  command?: string;
  result: string;
  isError: boolean;
}

export interface SystemNoticePayload {
  text: string;
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

export interface TurnStartedPayload {
  sessionId: string;
  contentFormat?: "markdown" | "html";
}

export interface TurnDonePayload {
  error?: string;
  cancelled?: boolean;
}

export interface SubagentToolBlock {
  id: string;
  name: string;
  args: string;
  command?: string;
  result?: string;
  running: boolean;
  isError?: boolean;
}

export interface SubagentRecord {
  id: string;
  label: string;
  prompt: string;
  agentType: string;
  background: boolean;
  status: "running" | "done" | "error";
  summary?: string;
  error?: string;
  collapsed?: boolean;
  tools: SubagentToolBlock[];
}

export type ChatBlock =
  | { id: string; role: "user"; text: string }
  | {
      id: string;
      role: "assistant";
      raw: string;
      contentFormat?: "markdown" | "html";
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
      mcpServer?: string;
    }
  | {
      id: string;
      role: "subagent";
      record: SubagentRecord;
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
  planning: boolean;
  contentFormat: "markdown" | "html";
  blocks: ChatBlock[];
  subagents: SubagentRecord[];
  usage: UsageUpdatePayload;
}

export const initialTurnState = (): TurnState => ({
  turnId: null,
  running: false,
  waitingPermission: false,
  planning: false,
  contentFormat: "markdown",
  blocks: [],
  subagents: [],
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

export function mcpServerFromToolName(name: string): string | undefined {
  if (!name.includes("__")) return undefined;
  const [server] = name.split("__", 2);
  return server || undefined;
}

export function turnReducer(state: TurnState, event: AgentEventEnvelope): TurnState {
  if (event.streamId !== "main" && event.streamId.startsWith("subagent:")) {
    return reduceSubagentStream(state, event);
  }

  switch (event.kind) {
    case "turn.started": {
      const payload = (event.payload ?? {}) as TurnStartedPayload;
      return {
        ...initialTurnState(),
        turnId: event.turnId,
        running: true,
        contentFormat: payload.contentFormat ?? "markdown",
      };
    }
    case "content.delta": {
      const payload = event.payload as ContentDeltaPayload;
      const blocks = [...state.blocks];
      const last = blocks[blocks.length - 1];
      if (last && last.role === "assistant" && last.streaming) {
        blocks[blocks.length - 1] = {
          ...last,
          raw: last.raw + payload.delta,
          reasoningOpen: false,
        };
      } else {
        blocks.push({
          id: nextBlockId("assistant"),
          role: "assistant",
          raw: payload.delta,
          contentFormat: state.contentFormat,
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
          contentFormat: state.contentFormat,
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
    case "planning.start":
      return { ...state, planning: true };
    case "planning.end":
      return { ...state, planning: false };
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
            mcpServer: mcpServerFromToolName(payload.name),
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
    case "subagent.start": {
      const p = event.payload as SubagentStartPayload;
      const record: SubagentRecord = {
        id: p.id,
        label: p.label,
        prompt: p.prompt,
        agentType: p.agentType,
        background: p.background,
        status: "running",
        collapsed: false,
        tools: [],
      };
      const subagents = [...state.subagents, record];
      const blocks = p.background
        ? state.blocks
        : [
            ...state.blocks,
            { id: nextBlockId("subagent"), role: "subagent" as const, record },
          ];
      return { ...state, subagents, blocks: syncSubagentBlocks(blocks, subagents) };
    }
    case "subagent.end": {
      const p = event.payload as SubagentEndPayload;
      const subagents = state.subagents.map((s) =>
        s.id === p.id
          ? {
              ...s,
              status: p.error ? ("error" as const) : ("done" as const),
              summary: p.summary,
              error: p.error,
            }
          : s,
      );
      return { ...state, subagents, blocks: syncSubagentBlocks(state.blocks, subagents) };
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
    case "system.notice": {
      const payload = event.payload as SystemNoticePayload;
      return {
        ...state,
        blocks: [...state.blocks, { id: nextBlockId("sys"), role: "system", text: payload.text }],
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
        planning: false,
        blocks: blocks.map((b) =>
          b.role === "assistant" && b.streaming ? { ...b, streaming: false } : b,
        ),
      };
    }
    default:
      return state;
  }
}

function reduceSubagentStream(state: TurnState, event: AgentEventEnvelope): TurnState {
  const subagentId = event.streamId.replace(/^subagent:/, "");
  if (event.kind === "subagent.tool.start") {
    const p = event.payload as SubagentToolStartPayload;
    const subagents = state.subagents.map((s) =>
      s.id === subagentId
        ? {
            ...s,
            tools: [
              ...s.tools,
              {
                id: nextBlockId("satool"),
                name: p.name,
                args: p.args,
                command: p.command,
                running: true,
              },
            ],
          }
        : s,
    );
    return { ...state, subagents };
  }
  if (event.kind === "subagent.tool.end") {
    const p = event.payload as SubagentToolEndPayload;
    const subagents = state.subagents.map((s) =>
      s.id === subagentId
        ? {
            ...s,
            tools: s.tools.map((t) =>
              t.name === p.name && t.running
                ? { ...t, running: false, result: p.result, isError: p.isError }
                : t,
            ),
          }
        : s,
    );
    return { ...state, subagents };
  }
  return state;
}

function syncSubagentBlocks(blocks: ChatBlock[], subagents: SubagentRecord[]): ChatBlock[] {
  return blocks.map((b) => {
    if (b.role !== "subagent") return b;
    const latest = subagents.find((s) => s.id === b.record.id);
    return latest ? { ...b, record: latest } : b;
  });
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
  contentFormat?: string;
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
      const fmt = m.contentFormat === "html" ? "html" : "markdown";
      blocks.push({
        id: nextBlockId("assistant"),
        role: "assistant",
        raw: m.content,
        contentFormat: fmt,
        reasoning: m.reasoning,
        streaming: false,
      });
    } else if (m.role === "tool") {
      const name = m.toolName || "tool";
      blocks.push({
        id: nextBlockId("tool"),
        role: "tool",
        name,
        args: m.toolCalls || m.content,
        result: m.content,
        running: false,
        collapsed: true,
        mcpServer: mcpServerFromToolName(name),
      });
    }
  }
  return blocks;
}

export const SLASH_COMMANDS = [
  { name: "help", description: "Show all slash commands" },
  { name: "compact", description: "Compact API context" },
  { name: "clear", description: "Start a new session" },
  { name: "context", description: "Token usage breakdown" },
  { name: "plan", description: "Enter Plan mode" },
  { name: "agent", description: "Return to Agent mode" },
  { name: "permissions", description: "View or switch permission mode" },
  { name: "git", description: "Refresh git snapshot" },
  { name: "mode", description: "Switch model" },
  { name: "resume", description: "Resume a session" },
];
