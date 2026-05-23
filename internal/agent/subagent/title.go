package subagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/billing"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/session/subagentstore"
	"github.com/hejunqiu/ds-code/internal/tool"
)

const (
	titleParentToolCallID = "session-title"
	titleRunLabel         = "Session title"
	titleThinkingType     = "disabled"
)

// GenerateSessionTitle runs a one-turn subagent to produce a session title.
// Prompt prefers Simplified Chinese; any non-empty model output is accepted.
func GenerateSessionTitle(ctx context.Context, cfg *config.Config, llmClient llm.Client, subStore subagentstore.Store, parentSessionID, userContent string) (string, error) {
	if parentSessionID == "" {
		return "", fmt.Errorf("subagent title: empty session id")
	}
	if subStore == nil {
		return "", fmt.Errorf("subagent title: store is required")
	}
	snippet := session.TitlePromptSnippet(userContent)
	if snippet == "" {
		return "", fmt.Errorf("subagent title: empty user content")
	}

	subModel := cfg.LLM.ResolveSubagentModel()
	run, err := getOrCreateTitleRun(ctx, subStore, parentSessionID, snippet, subModel, cfg)
	if err != nil {
		return "", err
	}

	if run.Status == subagentstore.StatusDone {
		if title, ok := titleFromRun(ctx, subStore, run.ID); ok {
			return title, nil
		}
	}

	prompt := fmt.Sprintf(`你是会话标题生成器。根据下面的用户首条消息，请用简体中文写一句不超过60字的标题。只输出标题正文，不要引号、解释或其它文字。若用户消息是英文，也请用中文概括。

用户消息：
%s`, snippet)

	ctx = agent.WithSessionTitleGen(ctx, false)
	raw, runErr := executeRun(ctx, cfg, llmClient, prompt, func(reg *tool.Registry) {}, subStore, run, nil, 1)
	status := subagentstore.StatusDone
	errMsg := ""
	if runErr != nil {
		status = subagentstore.StatusError
		errMsg = runErr.Error()
	}
	_ = subStore.FinishRun(ctx, run.ID, status, errMsg)
	if runErr != nil {
		return "", runErr
	}

	title := normalizeSessionTitle(raw)
	if title == "" {
		return "", fmt.Errorf("subagent title: empty model output")
	}
	return title, nil
}

func getOrCreateTitleRun(ctx context.Context, subStore subagentstore.Store, parentSessionID, snippet, subModel string, cfg *config.Config) (subagentstore.Run, error) {
	run, err := subStore.GetRunByToolCall(ctx, parentSessionID, titleParentToolCallID)
	if err == nil {
		return run, nil
	}
	return subStore.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID:     parentSessionID,
		ParentToolCallID:    titleParentToolCallID,
		RunKind:             subagentstore.RunKindTitle,
		Label:               titleRunLabel,
		Prompt:              snippet,
		Model:               subModel,
		ReasoningEffort:     cfg.LLM.ResolveSubagentReasoningEffort(),
		ThinkingType:        titleThinkingType,
		PricingSnapshotJSON: billing.MarshalSnapshot(billing.SnapshotForModel(subModel)),
	})
}

func titleFromRun(ctx context.Context, subStore subagentstore.Store, runID string) (string, bool) {
	msgs, err := subStore.ListMessages(ctx, runID)
	if err != nil {
		return "", false
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != role.Assistant {
			continue
		}
		title := normalizeSessionTitle(msgs[i].Content)
		if title != "" {
			return title, true
		}
	}
	return "", false
}

func normalizeSessionTitle(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'「」『』`)
	return session.TruncateTitle(s, session.MaxTitleRunes)
}
