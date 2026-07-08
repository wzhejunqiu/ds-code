import { useVirtualizer } from "@tanstack/react-virtual";
import { useEffect, useRef, useState } from "react";
import { PermissionCard } from "@/components/permission/PermissionCard";
import { SubagentCard } from "@/components/subagent/SubagentCard";
import { ToolCard } from "@/components/tools/ToolCard";
import { AssistantContent } from "@/render/AssistantContent";
import type { ChatBlock } from "@/protocol/agent-events";

function BlockView({
  block,
  workspaceId,
  onPermissionResolve,
  onToolToggle,
  onToolInspect,
  onSubagentToggle,
}: {
  block: ChatBlock;
  workspaceId: string;
  onPermissionResolve: (id: string, choice: string) => void;
  onToolToggle: (id: string) => void;
  onToolInspect?: (block: Extract<ChatBlock, { role: "tool" }>) => void;
  onSubagentToggle: (id: string) => void;
}) {
  switch (block.role) {
    case "user":
      return (
        <div className="mb-3 rounded-lg bg-[var(--color-muted)] px-3 py-2">
          <div className="mb-1 text-xs text-[var(--color-muted-foreground)]">You</div>
          <div className="whitespace-pre-wrap text-sm">{block.text}</div>
        </div>
      );
    case "assistant":
      return (
        <div className="mb-3">
          <div className="mb-1 text-xs text-[var(--color-muted-foreground)]">Assistant</div>
          {block.reasoning && (
            <details open={block.reasoningOpen} className="mb-2 text-xs text-[var(--color-muted-foreground)]">
              <summary>Thinking</summary>
              <pre className="mt-1 whitespace-pre-wrap">{block.reasoning}</pre>
            </details>
          )}
          <AssistantContent
            raw={block.raw}
            streaming={block.streaming}
            contentFormat={block.contentFormat}
          />
        </div>
      );
    case "tool":
      return (
        <ToolCard
          block={block}
          onToggle={() => onToolToggle(block.id)}
          onInspect={onToolInspect ? () => onToolInspect(block) : undefined}
        />
      );
    case "subagent":
      return (
        <SubagentCard
          record={block.record}
          onToggle={() => onSubagentToggle(block.id)}
        />
      );
    case "permission":
      return (
        <PermissionCard
          block={block}
          workspaceId={workspaceId}
          onResolve={onPermissionResolve}
        />
      );
    case "system":
      return (
        <div className="my-2 text-center text-xs text-[var(--color-muted-foreground)]">{block.text}</div>
      );
    default:
      return null;
  }
}

export function MessageList({
  blocks,
  workspaceId,
  onPermissionResolve,
  onToolToggle,
  onToolInspect,
  onSubagentToggle,
  follow,
}: {
  blocks: ChatBlock[];
  workspaceId: string;
  onPermissionResolve: (id: string, choice: string) => void;
  onToolToggle: (id: string) => void;
  onToolInspect?: (block: Extract<ChatBlock, { role: "tool" }>) => void;
  onSubagentToggle: (id: string) => void;
  follow: boolean;
}) {
  const parentRef = useRef<HTMLDivElement>(null);
  const [userScrolled, setUserScrolled] = useState(false);

  const virtualizer = useVirtualizer({
    count: blocks.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 120,
    overscan: 6,
  });

  useEffect(() => {
    if (follow && !userScrolled && blocks.length > 0) {
      virtualizer.scrollToIndex(blocks.length - 1, { align: "end" });
    }
  }, [blocks.length, follow, userScrolled, virtualizer]);

  return (
    <div
      ref={parentRef}
      className="min-h-0 flex-1 overflow-y-auto px-4 py-3"
      onScroll={() => setUserScrolled(true)}
    >
      <div style={{ height: virtualizer.getTotalSize(), position: "relative" }}>
        {virtualizer.getVirtualItems().map((item) => {
          const block = blocks[item.index];
          return (
            <div
              key={block.id}
              data-index={item.index}
              ref={virtualizer.measureElement}
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: "100%",
                transform: `translateY(${item.start}px)`,
              }}
            >
              <BlockView
                block={block}
                workspaceId={workspaceId}
                onPermissionResolve={onPermissionResolve}
                onToolToggle={onToolToggle}
                onToolInspect={onToolInspect}
                onSubagentToggle={onSubagentToggle}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
}
