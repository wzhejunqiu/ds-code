package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/agent/subagent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/register"
)

// TaskTool spawns read-only sub-agents for parallel exploration.
type TaskTool struct {
	Cfg      *config.Config
	Perm     *permission.Engine
	LLM      llm.Client
	Strict   bool
	Subagent subagentstore.Store

	sem taskSem
}

type taskSem struct {
	ch chan struct{}
}

func newTaskSemaphore(max int) taskSem {
	if max <= 0 {
		max = 3
	}
	return taskSem{ch: make(chan struct{}, max)}
}

func (s taskSem) Acquire(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s taskSem) Release() { <-s.ch }

// NewTaskTool creates a task tool with concurrency limit from config.
func NewTaskTool(cfg *config.Config, perm *permission.Engine, llmClient llm.Client, strict bool, sub subagentstore.Store) *TaskTool {
	return &TaskTool{
		Cfg:      cfg,
		Perm:     perm,
		LLM:      llmClient,
		Strict:   strict,
		Subagent: sub,
		sem:      newTaskSemaphore(cfg.Tools.Task.MaxParallel),
	}
}

func (t *TaskTool) Name() string { return "task" }

func (t *TaskTool) Description() string { return DescTask }

func (t *TaskTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"description": map[string]any{"type": "string", "description": SchemaTaskDescription},
		"prompt":      map[string]any{"type": "string", "description": SchemaTaskPrompt},
	}, []string{"prompt"}, t.Strict)
}

func (t *TaskTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *TaskTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Description string `json:"description"`
		Prompt      string `json:"prompt"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if in.Prompt == "" {
		return "", fmt.Errorf("%s", builtin.ErrPromptRequired)
	}

	if err := t.sem.Acquire(ctx); err != nil {
		return "", err
	}
	defer t.sem.Release()

	label := in.Description
	if label == "" {
		label = in.Prompt
	}

	inv, ok := agent.ToolInvocationFromContext(ctx)
	if !ok || inv.SessionID == "" || inv.ToolCallID == "" {
		return "", fmt.Errorf("%s", ErrMissingParent)
	}
	if t.Subagent == nil {
		return "", fmt.Errorf("%s", ErrNoSubStore)
	}

	subModel := t.Cfg.LLM.ResolveSubagentModel()
	run, err := t.Subagent.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID:     inv.SessionID,
		ParentToolCallID:    inv.ToolCallID,
		RunKind:             subagentstore.RunKindTask,
		Label:               label,
		Prompt:              in.Prompt,
		Model:               subModel,
		ReasoningEffort:     t.Cfg.LLM.ResolveSubagentReasoningEffort(),
		ThinkingType:        t.Cfg.LLM.ResolveSubagentThinkingType(),
		PricingSnapshotJSON: billing.MarshalSnapshot(billing.SnapshotForModel(subModel)),
	})
	if err != nil {
		return "", err
	}
	subID := run.ID

	parent := agent.TurnCallbacksFromContext(ctx)
	if parent != nil && parent.OnSubagentStart != nil {
		parent.OnSubagentStart(subID, label, in.Prompt)
	}

	gi, _ := tool.LoadGitignore(t.Cfg.ProjectRoot)
	subCB := agent.SubagentToolCallbacks(parent, subID)
	summary, runErr := subagent.Run(ctx, t.Cfg, t.LLM, in.Prompt, func(reg *tool.Registry) {
		register.ExploreTools(reg, t.Cfg, t.Perm, gi, t.Strict)
	}, t.Subagent, run, subCB, 0)

	status := subagentstore.StatusDone
	errMsg := ""
	if runErr != nil {
		status = subagentstore.StatusError
		errMsg = runErr.Error()
	}
	_ = t.Subagent.FinishRun(ctx, subID, status, errMsg)

	if parent != nil && parent.OnSubagentEnd != nil {
		parent.OnSubagentEnd(subID, summary, runErr)
	}
	if runErr != nil {
		return "", runErr
	}
	if in.Description != "" {
		return fmt.Sprintf("[%s]\n%s", in.Description, summary), nil
	}
	return summary, nil
}
