import { useEffect, useReducer, useRef, useState } from "react";
import { marked } from "marked";
import { Events } from "@wailsio/runtime";
import {
  initialTurnState,
  turnReducer,
  type AgentEventEnvelope,
  type ChatBlock,
  type TurnState,
} from "@/protocol/agent-events";
import { DesktopService } from "../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import "./App.css";

marked.setOptions({ breaks: true, gfm: true });

function renderMarkdown(raw: string): string {
  try {
    return marked.parse(raw, { async: false }) as string;
  } catch {
    return raw;
  }
}

type UIAction = AgentEventEnvelope | { type: "user"; text: string };

function uiReducer(state: TurnState, action: UIAction): TurnState {
  if ("type" in action && action.type === "user") {
    return {
      ...state,
      blocks: [
        ...state.blocks,
        { id: `user-${state.blocks.length + 1}`, role: "user", text: action.text },
      ],
    };
  }
  return turnReducer(state, action as AgentEventEnvelope);
}

function BlockView({ block }: { block: ChatBlock }) {
  switch (block.role) {
    case "user":
      return (
        <div className="msg msg-user">
          <div className="msg-label">You</div>
          <div className="msg-body">{block.text}</div>
        </div>
      );
    case "assistant":
      return (
        <div className="msg msg-assistant">
          <div className="msg-label">Assistant</div>
          {block.streaming ? (
            <pre className="msg-stream">{block.raw}</pre>
          ) : (
            <div
              className="msg-markdown"
              dangerouslySetInnerHTML={{ __html: renderMarkdown(block.raw) }}
            />
          )}
        </div>
      );
    case "tool":
      return (
        <div className="msg msg-tool">
          <div className="msg-label">
            Tool: {block.name}{" "}
            {block.running ? "(running)" : block.isError ? "(failed)" : "(done)"}
          </div>
          <pre className="msg-tool-body">{block.args}</pre>
          {block.result && (
            <pre className="msg-tool-result">{block.result.slice(0, 2000)}</pre>
          )}
        </div>
      );
    case "permission":
      if (block.resolved) {
        return (
          <div className="msg msg-permission resolved">
            Permission {block.allowed ? "granted" : "denied"} for {block.request.tool}
          </div>
        );
      }
      return (
        <div className="msg msg-permission">
          <div className="perm-title">Allow {block.request.tool}?</div>
          <pre className="perm-summary">{block.request.summary}</pre>
          <div className="perm-actions">
            <button
              type="button"
              onClick={() => DesktopService.ResolvePermission(block.request.id, true)}
            >
              Allow
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() => DesktopService.ResolvePermission(block.request.id, false)}
            >
              Deny
            </button>
          </div>
        </div>
      );
    case "system":
      return <div className="msg msg-system">{block.text}</div>;
    default:
      return null;
  }
}

export default function App() {
  const [input, setInput] = useState("");
  const [projectRoot, setProjectRoot] = useState("");
  const [status, setStatus] = useState("Open a project to begin.");
  const [turnState, dispatch] = useReducer(uiReducer, undefined, initialTurnState);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const off = Events.On("agent:event", (raw: { data: AgentEventEnvelope }) => {
      const event = raw.data;
      dispatch(event);
      if (event.kind === "permission.request") {
        setStatus("Waiting for permission approval…");
      }
      if (event.kind === "turn.done") {
        setStatus("Ready");
      }
    });
    return () => off?.();
  }, []);

  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight, behavior: "smooth" });
  }, [turnState.blocks]);

  const handleOpenProject = async () => {
    const root = projectRoot.trim() || ".";
    try {
      await DesktopService.OpenProject(root);
      setStatus(`Project opened: ${root}`);
    } catch (e) {
      setStatus(String(e));
    }
  };

  const handleSend = async () => {
    const text = input.trim();
    if (!text || turnState.running) return;
    setInput("");
    dispatch({ type: "user", text });
    setStatus("Running…");
    try {
      await DesktopService.SendMessage(text);
    } catch (e) {
      setStatus(String(e));
    }
  };

  const handleStop = async () => {
    try {
      await DesktopService.CancelTurn();
    } catch (e) {
      setStatus(String(e));
    }
  };

  return (
    <div className="app">
      <header className="header">
        <h1>ds-code Desktop</h1>
        <div className="project-row">
          <input
            aria-label="project root"
            placeholder="Project root path"
            value={projectRoot}
            onChange={(e) => setProjectRoot(e.target.value)}
          />
          <button type="button" onClick={handleOpenProject}>
            Open Project
          </button>
        </div>
        <div className="status">{status}</div>
      </header>

      <main className="chat" ref={listRef}>
        {turnState.blocks.map((block) => (
          <BlockView key={block.id} block={block} />
        ))}
      </main>

      <footer className="composer">
        <textarea
          aria-label="message"
          rows={3}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
              e.preventDefault();
              void handleSend();
            }
          }}
          placeholder="Message the agent (⌘Enter to send)"
        />
        {turnState.running ? (
          <button type="button" className="stop" onClick={handleStop}>
            Stop
          </button>
        ) : (
          <button type="button" onClick={handleSend} disabled={!input.trim()}>
            Send
          </button>
        )}
      </footer>
    </div>
  );
}
