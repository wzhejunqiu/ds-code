package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hejunqiu/ds-code/internal/agent/subagent"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/permission"
	toolpkg "github.com/hejunqiu/ds-code/internal/tool"
)

// TaskTool spawns read-only sub-agents for parallel exploration.
type TaskTool struct {
	Cfg   *config.Config
	Perm  *permission.Engine
	LLM   llm.Client
	Strict bool

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

func (s taskSem) Acquire() { s.ch <- struct{}{} }
func (s taskSem) Release() { <-s.ch }

// NewTaskTool creates a task tool with concurrency limit from config.
func NewTaskTool(cfg *config.Config, perm *permission.Engine, llmClient llm.Client, strict bool) *TaskTool {
	return &TaskTool{
		Cfg:    cfg,
		Perm:   perm,
		LLM:    llmClient,
		Strict: strict,
		sem:    newTaskSemaphore(cfg.Tools.Task.MaxParallel),
	}
}

func (t *TaskTool) Name() string { return "task" }

func (t *TaskTool) Description() string {
	return "Run a read-only sub-agent to explore the codebase in parallel. Returns a text summary."
}

func (t *TaskTool) Schema() map[string]any {
	return toolpkg.ObjectSchema(map[string]any{
		"description": map[string]any{"type": "string", "description": "Short label for this exploration"},
		"prompt":      map[string]any{"type": "string", "description": "Detailed instructions for the sub-agent"},
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
		return "", fmt.Errorf("prompt is required")
	}

	t.sem.Acquire()
	defer t.sem.Release()

	gi, _ := toolpkg.LoadGitignore(t.Cfg.ProjectRoot)
	summary, err := subagent.Run(ctx, t.Cfg, t.LLM, in.Prompt, func(reg *toolpkg.Registry) {
		RegisterExploreTools(reg, t.Cfg, t.Perm, gi, t.Strict)
	})
	if err != nil {
		return "", err
	}
	if in.Description != "" {
		return fmt.Sprintf("[%s]\n%s", in.Description, summary), nil
	}
	return summary, nil
}
