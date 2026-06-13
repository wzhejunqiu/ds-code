package manager_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/shelljobs"
	"github.com/wzhejunqiu/ds-code/internal/shelljobs/manager"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
)

func TestManager_backgroundJobCompletes(t *testing.T) {
	testutil.IsolatedHome(t)
	root := t.TempDir()
	mgr, err := manager.Open(root, config.ShellToolConfig{
		MaxBackground:            3,
		BackgroundOutputMaxBytes: 64 * 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	job, err := mgr.Start("echo hello-bg")
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		j, stdout, _, err := mgr.Get(job.ID, 4096)
		if err != nil {
			t.Fatal(err)
		}
		if j.Status == shelljobs.StatusCompleted {
			if !strings.Contains(stdout, "hello-bg") {
				t.Fatalf("stdout = %q", stdout)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for job, status=%s", j.Status)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestManager_cancelJob(t *testing.T) {
	testutil.IsolatedHome(t)
	root := t.TempDir()
	mgr, err := manager.Open(root, config.ShellToolConfig{MaxBackground: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	job, err := mgr.Start("sleep 60")
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
