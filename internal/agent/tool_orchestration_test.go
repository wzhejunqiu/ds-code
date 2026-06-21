package agent

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/shell"
)

func TestPartitionToolCalls_backgroundBashConcurrent(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Register(&shell.ShellTool{Strict: false})

	calls := []llm.ToolCall{
		{Name: "bash", Arguments: `{"command":"echo a","run_in_background":true}`},
		{Name: "bash", Arguments: `{"command":"echo b","run_in_background":true}`},
	}
	batches := partitionToolCalls(reg, calls)
	if len(batches) != 1 {
		t.Fatalf("batches = %d, want 1", len(batches))
	}
	if !batches[0].concurrent || len(batches[0].calls) != 2 {
		t.Fatalf("batch = %+v", batches[0])
	}
}

func TestPartitionToolCalls_syncBashSerial(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Register(&shell.ShellTool{Strict: false})

	calls := []llm.ToolCall{
		{Name: "bash", Arguments: `{"command":"echo sync"}`},
		{Name: "bash", Arguments: `{"command":"echo bg","run_in_background":true}`},
	}
	batches := partitionToolCalls(reg, calls)
	if len(batches) != 2 {
		t.Fatalf("batches = %d, want 2", len(batches))
	}
	if batches[0].concurrent || batches[0].calls[0].Arguments != `{"command":"echo sync"}` {
		t.Fatalf("first batch = %+v", batches[0])
	}
	if !batches[1].concurrent {
		t.Fatalf("second batch should be concurrent: %+v", batches[1])
	}
}
