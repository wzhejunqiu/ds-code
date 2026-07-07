import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { SLASH_COMMANDS } from "@/protocol/agent-events";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import { useAppState } from "@/state/app-store";

export function Composer({
  running,
  disabled,
  onSend,
  insertText,
  onInsertConsumed,
}: {
  running: boolean;
  disabled: boolean;
  onSend: (text: string) => void;
  insertText?: string;
  onInsertConsumed?: () => void;
}) {
  const { activeWorkspaceId, activeChatId, selectChat } = useAppState();
  const [input, setInput] = useState("");
  const [slashOpen, setSlashOpen] = useState(false);
  const [slashFilter, setSlashFilter] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (insertText) {
      setInput((prev) => (prev ? `${prev} ${insertText}` : insertText));
      onInsertConsumed?.();
    }
  }, [insertText, onInsertConsumed]);

  const send = async () => {
    const text = input.trim();
    if (!text || !activeWorkspaceId || !activeChatId) return;
    setInput("");
    setSlashOpen(false);
    onSend(text);
    const res = await DesktopService.SendMessage(activeWorkspaceId, activeChatId, text);
    if (res.handled && res.newSessionId) {
      await selectChat(res.newSessionId);
    }
  };

  const stop = async () => {
    if (!activeWorkspaceId) return;
    await DesktopService.CancelTurn(activeWorkspaceId);
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      void send();
    }
    if (e.key === "Escape") {
      setSlashOpen(false);
    }
  };

  const onChange = (value: string) => {
    setInput(value);
    if (value.startsWith("/")) {
      setSlashOpen(true);
      setSlashFilter(value.slice(1).toLowerCase());
    } else {
      setSlashOpen(false);
    }
  };

  const filtered = SLASH_COMMANDS.filter(
    (c) => c.name.includes(slashFilter) || c.description.toLowerCase().includes(slashFilter),
  );

  return (
    <div className="relative border-t border-[var(--color-border)] p-3">
      {slashOpen && filtered.length > 0 && (
        <div className="absolute bottom-full left-3 right-3 mb-1 max-h-40 overflow-auto rounded-md border border-[var(--color-border)] bg-[var(--color-card)] shadow-lg">
          {filtered.map((c) => (
            <button
              key={c.name}
              type="button"
              className="block w-full px-3 py-2 text-left text-sm hover:bg-[var(--color-muted)]"
              onClick={() => {
                setInput(`/${c.name} `);
                setSlashOpen(false);
                textareaRef.current?.focus();
              }}
            >
              <span className="font-medium">/{c.name}</span>
              <span className="ml-2 text-[var(--color-muted-foreground)]">{c.description}</span>
            </button>
          ))}
        </div>
      )}
      <textarea
        ref={textareaRef}
        className="mb-2 min-h-[72px] w-full resize-y rounded-md border border-[var(--color-border)] bg-[var(--color-background)] px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-[var(--color-primary)]"
        placeholder="Message… (⌘Enter to send, / for commands)"
        value={input}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={onKeyDown}
        disabled={disabled}
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault();
          const file = e.dataTransfer.files?.[0];
          if (file) {
            const path = (file as File & { path?: string }).path ?? file.name;
            setInput((prev) => `${prev}${prev ? " " : ""}@${path}`);
          }
        }}
      />
      <div className="flex justify-end">
        {running ? (
          <Button variant="destructive" onClick={() => void stop()}>
            Stop
          </Button>
        ) : (
          <Button onClick={() => void send()} disabled={disabled || !input.trim()}>
            Send
          </Button>
        )}
      </div>
    </div>
  );
}
