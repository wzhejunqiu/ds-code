export const PERMISSION_MODES = ["readonly", "ask", "auto"] as const;
export type PermissionMode = (typeof PERMISSION_MODES)[number];

export const LLM_MODELS = ["deepseek-v4-pro", "deepseek-v4-flash"] as const;
export const REASONING_EFFORTS = ["high", "max"] as const;
export const TRACING_EXPORTERS = ["", "log", "otlp"] as const;

export type SettingsSection =
  | "apiKey"
  | "model"
  | "permission"
  | "appearance"
  | "mcp"
  | "lsp"
  | "tracing"
  | "about";

export const SETTINGS_SECTIONS: { id: SettingsSection; label: string }[] = [
  { id: "apiKey", label: "API Key" },
  { id: "model", label: "Model" },
  { id: "permission", label: "Permissions" },
  { id: "appearance", label: "Appearance" },
  { id: "mcp", label: "MCP" },
  { id: "lsp", label: "LSP" },
  { id: "tracing", label: "Tracing" },
  { id: "about", label: "About" },
];

export const SETTINGS_SAVED_HINT =
  "Saved. New chats / restarted workspaces will use updated settings.";
