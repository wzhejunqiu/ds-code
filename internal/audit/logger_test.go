package audit_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/hejunqiu/ds-code/internal/audit"
)

func TestLogger_append(t *testing.T) {
	path := t.TempDir() + "/audit.jsonl"
	l := audit.NewLogger(path)
	if err := l.Log("grep", []byte(`{"pattern":"main"}`)); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var e struct {
		Tool     string `json:"tool"`
		ArgsHash string `json:"args_hash"`
	}
	if err := json.Unmarshal(b[:len(b)-1], &e); err != nil {
		t.Fatal(err)
	}
	if e.Tool != "grep" || len(e.ArgsHash) != 64 {
		t.Fatalf("entry = %+v", e)
	}
}
