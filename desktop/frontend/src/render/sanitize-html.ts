import DOMPurify from "dompurify";

const ALLOWED_TAGS = [
  "p",
  "br",
  "h1",
  "h2",
  "h3",
  "h4",
  "ul",
  "ol",
  "li",
  "table",
  "thead",
  "tbody",
  "tr",
  "th",
  "td",
  "pre",
  "code",
  "blockquote",
  "strong",
  "em",
  "a",
  "details",
  "summary",
  "span",
  "div",
];

const ALLOWED_ATTR = ["href", "class", "id", "title", "target", "rel"];

let hooksInstalled = false;

function installHooks() {
  if (hooksInstalled || typeof window === "undefined") return;
  hooksInstalled = true;
  DOMPurify.addHook("afterSanitizeAttributes", (node) => {
    if (node.tagName === "A") {
      const href = node.getAttribute("href") ?? "";
      if (href.startsWith("https:") || href.startsWith("mailto:")) {
        node.setAttribute("target", "_blank");
        node.setAttribute("rel", "noopener noreferrer");
      } else {
        node.removeAttribute("href");
      }
    }
  });
}

export function sanitizeAssistantHtml(dirty: string): string {
  installHooks();
  return DOMPurify.sanitize(dirty, {
    ALLOWED_TAGS,
    ALLOWED_ATTR,
    ALLOW_DATA_ATTR: false,
  });
}

export { ALLOWED_TAGS, ALLOWED_ATTR };
