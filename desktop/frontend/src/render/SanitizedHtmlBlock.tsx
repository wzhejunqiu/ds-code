import { useEffect, useRef } from "react";
import { sanitizeAssistantHtml } from "@/render/sanitize-html";

const shadowStyles = `
:host {
  display: block;
  color: inherit;
  font-size: 0.875rem;
  line-height: 1.5;
}
a { color: #60a5fa; }
table { border-collapse: collapse; width: 100%; margin: 0.5rem 0; }
th, td { border: 1px solid #444; padding: 0.25rem 0.5rem; text-align: left; }
pre { overflow-x: auto; padding: 0.5rem; background: rgba(255,255,255,0.05); border-radius: 0.25rem; }
code { font-family: ui-monospace, monospace; font-size: 0.85em; }
blockquote { border-left: 3px solid #666; margin: 0.5rem 0; padding-left: 0.75rem; opacity: 0.9; }
details { margin: 0.5rem 0; }
`;

export function SanitizedHtmlBlock({ html }: { html: string }) {
  const hostRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    let shadow = host.shadowRoot;
    if (!shadow) {
      shadow = host.attachShadow({ mode: "open" });
      const style = document.createElement("style");
      style.textContent = shadowStyles;
      shadow.appendChild(style);
      const container = document.createElement("div");
      container.className = "assistant-html";
      shadow.appendChild(container);
    }
    const container = shadow.querySelector(".assistant-html");
    if (container) {
      container.innerHTML = sanitizeAssistantHtml(html);
    }
  }, [html]);

  return <div ref={hostRef} className="assistant-html-host" />;
}
