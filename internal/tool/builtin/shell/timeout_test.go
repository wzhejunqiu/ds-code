package shell

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

func TestResolveShellTimeout_usesConfigDefault(t *testing.T) {
	cfg := &config.Config{Tools: config.ToolsConfig{Shell: config.ShellToolConfig{Timeout: 45 * time.Second}}}
	d, err := ResolveShellTimeout(cfg, 0)
	if err != nil {
		t.Fatal(err)
	}
	if d != 45*time.Second {
		t.Fatalf("got %v, want 45s", d)
	}
}

func TestResolveShellTimeout_defaultWhenConfigZero(t *testing.T) {
	cfg := &config.Config{Tools: config.ToolsConfig{Shell: config.ShellToolConfig{Timeout: 0}}}
	d, err := ResolveShellTimeout(cfg, 0)
	if err != nil {
		t.Fatal(err)
	}
	if d != 120*time.Second {
		t.Fatalf("got %v, want default 120s", d)
	}
}

func TestResolveShellTimeout_explicitMs(t *testing.T) {
	cfg := &config.Config{Tools: config.ToolsConfig{Shell: config.ShellToolConfig{Timeout: time.Minute}}}
	d, err := ResolveShellTimeout(cfg, 1500)
	if err != nil {
		t.Fatal(err)
	}
	if d != 1500*time.Millisecond {
		t.Fatalf("got %v, want 1500ms", d)
	}
}

func TestResolveShellTimeout_negativeRejected(t *testing.T) {
	cfg := &config.Config{}
	_, err := ResolveShellTimeout(cfg, -1)
	if err == nil {
		t.Fatal("expected error for negative timeout_ms")
	}
}

func TestShellTimeoutDeadline_emptyCommand(t *testing.T) {
	cfg := &config.Config{Tools: config.ToolsConfig{Shell: config.ShellToolConfig{Timeout: time.Minute}}}
	now := time.Now()

	for _, raw := range [][]byte{
		[]byte(`{"command":""}`),
		[]byte(`{"command":"   "}`),
		[]byte(`{}`),
	} {
		deadline, ok := shellTimeoutDeadline(now, cfg, raw)
		if ok || !deadline.IsZero() {
			t.Fatalf("empty command should not produce deadline: deadline=%v ok=%v args=%s", deadline, ok, raw)
		}
	}
}

func TestShellTimeoutDeadline_nilConfig(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"command": "echo hi"})
	deadline, ok := shellTimeoutDeadline(time.Now(), nil, raw)
	if ok || !deadline.IsZero() {
		t.Fatalf("nil cfg should not produce deadline: deadline=%v ok=%v", deadline, ok)
	}
}

func TestShellTimeoutDeadline_negativeTimeout(t *testing.T) {
	cfg := &config.Config{Tools: config.ToolsConfig{Shell: config.ShellToolConfig{Timeout: time.Minute}}}
	raw, _ := json.Marshal(map[string]any{"command": "echo hi", "timeout_ms": -1})
	deadline, ok := shellTimeoutDeadline(time.Now(), cfg, raw)
	if ok || !deadline.IsZero() {
		t.Fatalf("negative timeout_ms should not produce deadline: deadline=%v ok=%v", deadline, ok)
	}
}

func TestShellTimeoutDeadline_respectsTimeoutMs(t *testing.T) {
	cfg := &config.Config{Tools: config.ToolsConfig{Shell: config.ShellToolConfig{Timeout: time.Minute}}}
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	raw, _ := json.Marshal(map[string]any{"command": "echo hi", "timeout_ms": 5000})

	deadline, ok := shellTimeoutDeadline(now, cfg, raw)
	if !ok {
		t.Fatal("expected deadline")
	}
	want := now.Add(5 * time.Second)
	if !deadline.Equal(want) {
		t.Fatalf("deadline = %v, want %v", deadline, want)
	}
}

func TestShellSyncTimeoutDeadline_alias(t *testing.T) {
	cfg := &config.Config{Tools: config.ToolsConfig{Shell: config.ShellToolConfig{Timeout: 10 * time.Second}}}
	now := time.Now()
	raw, _ := json.Marshal(map[string]any{"command": "echo hi"})

	deadline, ok := ShellSyncTimeoutDeadline(now, cfg, raw)
	if !ok || !deadline.Equal(now.Add(10*time.Second)) {
		t.Fatalf("ShellSyncTimeoutDeadline = %v ok=%v", deadline, ok)
	}
}

func TestTimeoutMsFromArgs_coercesNumericTypes(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		want int
	}{
		{"float64", map[string]any{"timeout_ms": float64(3000)}, 3000},
		{"int", map[string]any{"timeout_ms": 4000}, 4000},
		{"int64", map[string]any{"timeout_ms": int64(5000)}, 5000},
		{"json.Number", map[string]any{"timeout_ms": json.Number("6000")}, 6000},
		{"missing", map[string]any{}, 0},
		{"wrong type", map[string]any{"timeout_ms": "5000"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timeoutMsFromArgs(tt.args); got != tt.want {
				t.Fatalf("timeoutMsFromArgs() = %d, want %d", got, tt.want)
			}
		})
	}
}
