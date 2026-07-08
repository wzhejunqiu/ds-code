export type ContentFormat = "markdown" | "html";

export function looksLikeMarkdown(raw: string): boolean {
  const trimmed = raw.trim();
  if (!trimmed.includes("<")) return true;
  if (/^#{1,6}\s/m.test(trimmed)) return true;
  if (/```/.test(trimmed)) return true;
  return false;
}

export function resolveRenderedFormat(
  contentFormat: ContentFormat | undefined,
  raw: string,
): { format: ContentFormat; fallback: boolean } {
  const fmt = contentFormat ?? "markdown";
  if (fmt === "html" && looksLikeMarkdown(raw)) {
    return { format: "markdown", fallback: true };
  }
  return { format: fmt, fallback: false };
}
