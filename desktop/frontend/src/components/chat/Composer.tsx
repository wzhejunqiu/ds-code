import { useState } from "react";
import { Button } from "@/components/ui/button";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import { useAppState } from "@/state/app-store";

export function Composer({
  running,
  disabled,
  onSend,
}: {
  running: boolean;
  disabled: boolean;
  onSend: (text: string) => void;
}) {
  const { activeWorkspaceId, activeChatId } = useAppState();
  const [input, setInput] = useState("");

  const send = async () => {
    const text = input.trim();
    if (!text || !activeWorkspaceId || !activeChatId) return;
    setInput("");
    onSend(text);
    await DesktopService.SendMessage(activeWorkspaceId, activeChatId, text);
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
  };

  return (
    <div className="border-t border-[var(--color-border)] p-3">
      <textarea
        className="mb-2 min-h-[72px] w-full resize-y rounded-md border border-[var(--color-border)] bg-[var(--color-background)] px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-[var(--color-primary)]"
        placeholder="Message… (⌘Enter to send)"
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={onKeyDown}
        disabled={disabled}
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
