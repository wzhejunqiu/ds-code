package spawn

import (
	"context"
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/mcp/resultstore"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"go.uber.org/zap"
)

// BackgroundManager tracks in-flight async agents and drains completion notifications.
type BackgroundManager struct {
	mu     sync.Mutex
	tasks  map[string]*BackgroundTask
	notify *NotificationQueue
}

// BackgroundTask is one running async agent.
type BackgroundTask struct {
	RunID     string
	AgentType AgentType
	Label     string
	StartTime time.Time
	Cancel    context.CancelFunc
	Done      chan struct{}
}

// NewBackgroundManager creates a manager backed by a notification queue.
func NewBackgroundManager(nq *NotificationQueue) *BackgroundManager {
	return &BackgroundManager{
		tasks:  make(map[string]*BackgroundTask),
		notify: nq,
	}
}

// Start launches an agent run in a background goroutine.
func (bm *BackgroundManager) Start(parentCtx context.Context, cfg *config.Config, llmClient llm.Client, run subagentstore.Run, def AgentTypeDefinition, perm *permission.Engine, parentReg *tool.Registry, subStore subagentstore.Store, parentCallbacks *agent.TurnCallbacks, hooks *agent.HookManager, failCleanup func(context.Context, subagentstore.Run), mcpResults *resultstore.Store) {
	ctx, cancel := DetachSpawnContext(parentCtx)
	task := &BackgroundTask{
		RunID:     run.ID,
		AgentType: def.Type,
		Label:     run.Label,
		StartTime: time.Now(),
		Cancel:    cancel,
		Done:      make(chan struct{}),
	}

	bm.mu.Lock()
	bm.tasks[run.ID] = task
	bm.mu.Unlock()

	go func() {
		defer close(task.Done)
		defer cancel()

		startTime := time.Now()
		cb := agent.SubagentToolCallbacks(parentCallbacks, run.ID)
		summary, runErr := ExecuteRun(ctx, cfg, llmClient, run, def, perm, parentReg, subStore, cb, hooks, 0, mcpResults)

		status := subagentstore.StatusCompleted
		errMsg := ""
		if runErr != nil {
			if ctx.Err() != nil {
				status = subagentstore.StatusKilled
			} else {
				status = subagentstore.StatusError
			}
			errMsg = runErr.Error()
		}
		durationMS := time.Since(startTime).Milliseconds()

		if finishErr := subStore.FinishRun(ctx, run.ID, status, errMsg); finishErr != nil {
			logging.L().Warn("finish async agent failed", zap.String("run_id", run.ID), zap.Error(finishErr))
		}
		if hooks != nil {
			in := agent.HookInput{AgentID: run.ID, AgentType: def.Type.String()}
			if runErr != nil {
				in.Error = runErr.Error()
			}
			hooks.Run(ctx, agent.HookSubagentStop, agent.MarshalHookInput(in))
		}
		if failCleanup != nil && (runErr != nil || status == subagentstore.StatusKilled) {
			failCleanup(ctx, run)
		}

		resultStatus, err := resultStatusFromStore(status)
		if err != nil {
			logging.L().Error("map subagent result status", zap.String("run_id", run.ID), zap.Error(err))
			return
		}
		summaryText := agentSummaryText(run.Label, resultStatus, runErr)
		delivered := DeliverResult(cfg.ProjectDataDir, run.ParentSessionID, run.ParentToolCallID, summary, resultStatus, runErr, cfg)
		n := Notification{
			AgentID:        run.ID,
			ToolUseID:      run.ParentToolCallID,
			Status:         resultStatus,
			Summary:        summaryText,
			DurationMS:     durationMS,
			ToolUseCount:   countToolUses(ctx, subStore, run.ID),
			WorktreePath:   run.WorktreePath,
			WorktreeBranch: run.WorktreeBranch,
		}
		if delivered.Inline {
			n.Result = delivered.Body
		} else {
			n.OutputFile = delivered.OutputPath
		}

		bm.notify.Enqueue(n, notificationPriority(parentCtx))

		bm.mu.Lock()
		delete(bm.tasks, run.ID)
		bm.mu.Unlock()

		logging.L().Info("async agent finished",
			zap.String("run_id", run.ID),
			zap.String("status", resultStatus.String()),
			zap.Int64("duration_ms", durationMS),
		)
	}()
}

// Kill cancels a running background agent.
func (bm *BackgroundManager) Kill(runID string) {
	bm.mu.Lock()
	task, ok := bm.tasks[runID]
	bm.mu.Unlock()
	if ok {
		task.Cancel()
	}
}

// RunningCount returns the number of in-flight agents.
func (bm *BackgroundManager) RunningCount() int {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return len(bm.tasks)
}

// List returns all currently running tasks.
func (bm *BackgroundManager) List() []BackgroundTask {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	var out []BackgroundTask
	for _, t := range bm.tasks {
		out = append(out, *t)
	}
	return out
}
