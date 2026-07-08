import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { DesktopService } from "../../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";

export function AppearancePanel() {
  const [outputFormat, setOutputFormat] = useState<"markdown" | "html">("markdown");
  const [htmlAck, setHtmlAck] = useState(() => localStorage.getItem("ds-code-html-ack") === "1");
  const [msg, setMsg] = useState("");

  const loadDesktopPrefs = useCallback(async () => {
    const cfg = await DesktopService.GetConfig("user", "");
    setOutputFormat(cfg.assistantOutputFormat === "html" ? "html" : "markdown");
  }, []);

  useEffect(() => {
    void loadDesktopPrefs();
  }, [loadDesktopPrefs]);

  const saveOutputFormat = async (format: "markdown" | "html") => {
    if (format === "html" && !htmlAck) {
      const ok = window.confirm(
        "HTML 模式会渲染模型输出的富文本。内容经 DOMPurify 消毒，但仍可能产生误导性排版。是否启用？",
      );
      if (!ok) return;
      localStorage.setItem("ds-code-html-ack", "1");
      setHtmlAck(true);
    }
    setMsg("");
    try {
      await DesktopService.SaveDesktopAssistantOutputFormat(format);
      setOutputFormat(format);
      setMsg("Assistant output format saved (default for new sessions).");
    } catch (e) {
      setMsg(String(e));
    }
  };

  return (
    <section>
      <h3 className="mb-2 text-sm font-medium">Appearance · Assistant output</h3>
      <p className="mb-2 text-xs text-[var(--color-muted-foreground)]">
        新会话默认格式。会话内可在聊天区 Output 切换（仅影响后续回复）。
      </p>
      <div className="flex flex-wrap gap-2">
        {(["markdown", "html"] as const).map((f) => (
          <Button
            key={f}
            variant={outputFormat === f ? "default" : "secondary"}
            size="sm"
            onClick={() => void saveOutputFormat(f)}
          >
            {f}
          </Button>
        ))}
      </div>
      {msg && <p className="mt-2 text-xs text-[var(--color-muted-foreground)]">{msg}</p>}
    </section>
  );
}
