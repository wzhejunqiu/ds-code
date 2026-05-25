package spawn

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"go.uber.org/zap"
)

// BackgroundManager tracks in-flight async agents and drains completion notifications.
type BackgroundManager struct {
	mu       sync.Mutex
	tasks    map[string]*BackgroundTask
	notify   *NotificationQueue
}

// BackgroundTask is one running async agent.
type BackgroundTask struct {
	RunID      string
	AgentType  string
	Label      string
	StartTime  time.Time
	Cancel     context.CancelFunc
	Done       chan struct{}
}

// NewBackgroundManager creates a manager backed by a notification queue.
func NewBackgroundManager(nq *NotificationQueue) *BackgroundManager {
	return &BackgroundManager{
		tasks:  make(map[string]*BackgroundTask),
		notify: nq,
	}
}

// Start launches an agent run in a background goroutine.
func (bm *BackgroundManager) Start(parentCtx context.Context, cfg *config.Config, llmClient llm.Client, run subagentstore.Run, def AgentTypeDefinition, perm *permission.Engine, parentReg *tool.Registry, subStore subagentstore.Store, parentCallbacks *agent.TurnCallbacks, outputPath string, hooks *agent.HookManager, failCleanup func(context.Context, subagentstore.Run)) {
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

		// Ensure output directory exists
		if outputPath != "" {
			if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
				logging.L().Warn("create output dir failed", zap.String("run_id", run.ID), zap.Error(err))
			}
		}

		startTime := time.Now()
		cb := agent.SubagentToolCallbacks(parentCallbacks, run.ID)
		summary, runErr := ExecuteRun(ctx, cfg, llmClient, run, def, perm, parentReg, subStore, cb, hooks, 0)

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
			in := agent.HookInput{AgentID: run.ID, AgentType: def.Type}
			if runErr != nil {
				in.Error = runErr.Error()
			}
			hooks.Run(ctx, agent.HookSubagentStop, agent.MarshalHookInput(in))
		}
		if failCleanup != nil && (runErr != nil || status == subagentstore.StatusKilled) {
			failCleanup(ctx, run)
		}

		// Build notification
		statusStr := "completed"
		switch status {
		case subagentstore.StatusError:
			statusStr = "failed"
		case subagentstore.StatusKilled:
			statusStr = "killed"
		}

		summaryText := fmt.Sprintf("Agent %q %s", run.Label, statusStr)
		if runErr != nil {
			summaryText = fmt.Sprintf("Agent %q failed: %s", run.Label, runErr.Error())
		}

		bm.notify.Enqueue(Notification{
			AgentID:        run.ID,
			ToolUseID:      run.ParentToolCallID,
			OutputFile:     outputPath,
			Status:         statusStr,
			Summary:        summaryText,
			Result:         summary,
			DurationMS:     durationMS,
			ToolUseCount:   countToolUses(ctx, subStore, run.ID),
			WorktreePath:   run.WorktreePath,
			WorktreeBranch: run.WorktreeBranch,
		}, notificationPriority(parentCtx))

		// Write output file
		if outputPath != "" {
			writeOutputFile(outputPath, summary, statusStr, runErr)
		}

		bm.mu.Lock()
		delete(bm.tasks, run.ID)
		bm.mu.Unlock()

		logging.L().Info("async agent finished",
			zap.String("run_id", run.ID),
			zap.String("status", statusStr),
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

// RegisterPromoted records a sync agent that was promoted to background without starting a new goroutine.
func (bm *BackgroundManager) RegisterPromoted(runID, agentType, label string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.tasks[runID] = &BackgroundTask{
		RunID:     runID,
		AgentType: agentType,
		Label:     label,
		StartTime: time.Now(),
		Done:      make(chan struct{}),
	}
}

// CompletePromoted marks a promoted agent finished and removes it from the running set.
func (bm *BackgroundManager) CompletePromoted(runID string) {
	bm.mu.Lock()
	task, ok := bm.tasks[runID]
	if ok {
		close(task.Done)
		delete(bm.tasks, runID)
	}
	bm.mu.Unlock()
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

func writeOutputFile(path, summary, status string, err error) {
	f, fileErr := os.Create(path)
	if fileErr != nil {
		logging.L().Warn("cannot create output file", zap.String("path", path), zap.Error(fileErr))
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "status: %s\n", status)
	if err != nil {
		fmt.Fprintf(f, "error: %s\n", err.Error())
	}
	fmt.Fprintf(f, "\n%s\n", summary)
}
