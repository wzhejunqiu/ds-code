package manager_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/shelljobs"
	"github.com/wzhejunqiu/ds-code/internal/shelljobs/manager"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
)

func openTestManager(t *testing.T, cfg config.ShellToolConfig) (*manager.Manager, string) {
	t.Helper()
	root := testutil.IsolatedProjectRoot(t)
	mgr, err := manager.Open(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		mgr.Close()
		if err := testutil.CleanupProjectData(root); err != nil {
			t.Errorf("cleanup project data: %v", err)
		}
	})
	return mgr, root
}

func TestManager_backgroundJobCompletes(t *testing.T) {
	mgr, _ := openTestManager(t, config.ShellToolConfig{
		MaxBackground:            3,
		BackgroundOutputMaxBytes: 64 * 1024,
	})

	job, err := mgr.Start("echo hello-bg", "Echo test")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	finished, err := mgr.Wait(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != shelljobs.StatusCompleted {
		t.Fatalf("status = %s", finished.Status)
	}
	_, stdout, _, err := mgr.Get(job.ID, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "hello-bg") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestManager_waitContextCancel(t *testing.T) {
	mgr, _ := openTestManager(t, config.ShellToolConfig{MaxBackground: 2})

	job, err := mgr.Start("sleep 60", "Long sleep")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, err := mgr.Wait(ctx, job.ID); err == nil {
		t.Fatal("expected wait error on cancel")
	}
	j, _, _, err := mgr.Get(job.ID, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if j.Status != shelljobs.StatusKilled {
		t.Fatalf("status = %s, want killed", j.Status)
	}
}

func TestManager_cancelJob(t *testing.T) {
	mgr, _ := openTestManager(t, config.ShellToolConfig{MaxBackground: 2})

	job, err := mgr.Start("sleep 60", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = mgr.Cancel(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	j, _, _, err := mgr.Get(job.ID, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if j.Status != shelljobs.StatusKilled {
		t.Fatalf("status = %s", j.Status)
	}
}

func TestManager_CloseKillsRunningJobs(t *testing.T) {
	mgr, _ := openTestManager(t, config.ShellToolConfig{MaxBackground: 2})

	job, err := mgr.Start("sleep 120", "Long task")
	if err != nil {
		t.Fatal(err)
	}
	mgr.Close()

	j, _, _, err := mgr.Get(job.ID, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if j.Status != shelljobs.StatusKilled {
		t.Fatalf("status = %s, want killed", j.Status)
	}
}

func TestManager_reconcileStaleJobs(t *testing.T) {
	root := testutil.IsolatedProjectRoot(t)
	cfg := config.ShellToolConfig{MaxBackground: 2}

	if _, err := config.EnsureProjectDataDir(root); err != nil {
		t.Fatal(err)
	}
	jobsDir := config.DefaultShellJobsDir(root)
	id := "stale01"
	jobDir := filepath.Join(jobsDir, id)
	if err := os.MkdirAll(jobDir, 0o700); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sleep", "120")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_, _ = cmd.Process.Wait()
	})

	job := shelljobs.Job{
		ID:         id,
		Command:    "sleep 120",
		PID:        cmd.Process.Pid,
		Status:     shelljobs.StatusRunning,
		StartedAt:  time.Now().UTC(),
		StdoutPath: filepath.Join(jobDir, "stdout.log"),
		StderrPath: filepath.Join(jobDir, "stderr.log"),
	}
	meta, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jobDir, "meta.json"), meta, 0o600); err != nil {
		t.Fatal(err)
	}

	mgr, err := manager.Open(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		mgr.Close()
		if err := testutil.CleanupProjectData(root); err != nil {
			t.Errorf("cleanup project data: %v", err)
		}
	})

	deadline := time.Now().Add(5 * time.Second)
	for {
		j, _, _, err := mgr.Get(id, 1024)
		if err != nil {
			t.Fatal(err)
		}
		if j.Status == shelljobs.StatusKilled {
			waitDone := make(chan struct{})
			go func() {
				_ = cmd.Wait()
				close(waitDone)
			}()
			select {
			case <-waitDone:
				return
			case <-time.After(2 * time.Second):
				t.Fatalf("stale pid %d not reaped after reconcile", job.PID)
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("stale job status = %s, want killed (reconcile timeout)", j.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
