import { useAppState } from "@/state/app-store";

export function ApiKeyPanel() {
  const { apiKeyOk, apiKeyHint } = useAppState();

  return (
    <section>
      <h3 className="mb-2 text-sm font-medium">API Key</h3>
      <p className="text-sm text-[var(--color-muted-foreground)]">
        {apiKeyOk
          ? "API key detected via environment variables."
          : `Not configured. Set DS_CODE_DEEPSEEK_API_KEY or DEEPSEEK_API_KEY. ${apiKeyHint}`}
      </p>
      <p className="mt-3 text-xs text-[var(--color-muted-foreground)]">
        API keys are read from the environment only and are not stored in config files.
      </p>
    </section>
  );
}
