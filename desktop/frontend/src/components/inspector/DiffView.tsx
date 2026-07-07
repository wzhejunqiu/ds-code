import { DiffEditor } from "@monaco-editor/react";
import { useEffect, useState } from "react";
import { DesktopService } from "../../../bindings/github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop";
import { useInspector } from "@/state/inspector-store";

export function DiffView({ workspaceId }: { workspaceId: string }) {
  const { selection, diffInline } = useInspector();
  const [files, setFiles] = useState<
    { path: string; original: string; modified: string; language: string }[]
  >([]);
  const [fileIndex, setFileIndex] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (selection.kind !== "diff" || !workspaceId) return;
    setLoading(true);
    setError("");
    void (async () => {
      try {
        const diffs = await DesktopService.PreviewPatch(workspaceId, selection.patchText);
        setFiles(diffs ?? []);
        setFileIndex(selection.fileIndex ?? 0);
      } catch (e) {
        setError(String(e));
        setFiles([]);
      } finally {
        setLoading(false);
      }
    })();
  }, [selection, workspaceId]);

  if (selection.kind === "file") {
    return <FilePreviewView workspaceId={workspaceId} path={selection.path} offset={selection.offset} limit={selection.limit} />;
  }

  if (loading) return <div className="p-4 text-sm text-[var(--color-muted-foreground)]">Loading diff…</div>;
  if (error) return <div className="p-4 text-sm text-red-400">{error}</div>;
  if (files.length === 0) return <div className="p-4 text-sm text-[var(--color-muted-foreground)]">No diff to show.</div>;

  const f = files[Math.min(fileIndex, files.length - 1)];
  return (
    <div className="flex h-full min-h-0 flex-col">
      {files.length > 1 && (
        <div className="flex flex-wrap gap-1 border-b border-[var(--color-border)] p-2">
          {files.map((file, i) => (
            <button
              key={file.path}
              type="button"
              className={`rounded px-2 py-1 text-xs ${i === fileIndex ? "bg-[var(--color-primary)] text-white" : "bg-[var(--color-muted)]"}`}
              onClick={() => setFileIndex(i)}
            >
              {file.path}
            </button>
          ))}
        </div>
      )}
      <div className="min-h-0 flex-1">
        <DiffEditor
          height="100%"
          language={f.language}
          original={f.original}
          modified={f.modified}
          options={{
            readOnly: true,
            renderSideBySide: !diffInline,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
          }}
          theme="vs-dark"
        />
      </div>
    </div>
  );
}

function FilePreviewView({
  workspaceId,
  path,
  offset,
  limit,
}: {
  workspaceId: string;
  path: string;
  offset?: number;
  limit?: number;
}) {
  const [content, setContent] = useState("");
  const [language, setLanguage] = useState("plaintext");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void (async () => {
      setLoading(true);
      const res = await DesktopService.ReadFilePreview(workspaceId, path, offset ?? 0, limit ?? 0);
      setContent(res.content);
      setLanguage(res.language);
      setLoading(false);
    })();
  }, [workspaceId, path, offset, limit]);

  if (loading) return <div className="p-4 text-sm">Loading…</div>;

  return (
    <div className="h-full min-h-0">
      <DiffEditor
        height="100%"
        language={language}
        original={content}
        modified={content}
        options={{
          readOnly: true,
          renderSideBySide: false,
          minimap: { enabled: false },
          lineNumbers: "on",
        }}
        theme="vs-dark"
      />
    </div>
  );
}
