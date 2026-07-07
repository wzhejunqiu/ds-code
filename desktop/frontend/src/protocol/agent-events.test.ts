import { describe, expect, it, beforeEach } from "vitest";
import {
  initialTurnState,
  turnReducer,
  type AgentEventEnvelope,
} from "./agent-events";

function env(kind: AgentEventEnvelope["kind"], payload: unknown, turnId = "t1"): AgentEventEnvelope {
  return {
    v: 1,
    seq: 1,
    turnId,
    streamId: "main",
    workspaceId: "default",
    kind,
    ts: 0,
    critical: kind === "turn.done",
    payload,
  };
}

describe("turnReducer", () => {
  beforeEach(() => {
    // reset block ids by re-importing module state — use fresh initial state each test
  });

  it("streams assistant content and completes turn", () => {
    let state = initialTurnState();
    state = turnReducer(state, env("turn.started", { sessionId: "s1" }));
    state = turnReducer(state, env("content.delta", { delta: "Hello" }));
    state = turnReducer(state, env("assistant.segment_end", {}));
    state = turnReducer(state, env("turn.done", {}));

    expect(state.running).toBe(false);
    expect(state.blocks.filter((b) => b.role === "assistant")).toHaveLength(1);
    expect(state.blocks.find((b) => b.role === "assistant")?.raw).toBe("Hello");
  });

  it("records permission request block", () => {
    let state = initialTurnState();
    state = turnReducer(
      state,
      env("permission.request", {
        id: "p1",
        kind: "write_shell",
        tool: "apply_patch",
        summary: "edit foo.go",
      }),
    );
    expect(state.waitingPermission).toBe(true);
    expect(state.blocks.some((b) => b.role === "permission")).toBe(true);
  });
});
