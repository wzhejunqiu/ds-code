package agent

import (
	"testing"
)

func TestApplyPreToolUseResults_shellCommand(t *testing.T) {
	raw := []byte(`{"command":"pip install foo"}`)
	results := []HookResult{{Output: `{"command":"uv pip install foo"}`}}
	out := applyPreToolUseResults("bash", raw, results)
	if string(out) != `{"command":"uv pip install foo"}` {
		t.Fatalf("got %s", string(out))
	}
}

func TestApplyPreToolUseResults_argsMerge(t *testing.T) {
	raw := []byte(`{"filepath":"a.go"}`)
	results := []HookResult{{Output: `{"args":{"offset":10}}`}}
	out := applyPreToolUseResults("read_file", raw, results)
	if string(out) != `{"offset":10,"filepath":"a.go"}` && string(out) != `{"filepath":"a.go","offset":10}` {
		t.Fatalf("got %s", string(out))
	}
}

func TestApplyPreToolUseResults_invalidJSONIgnored(t *testing.T) {
	raw := []byte(`{"command":"ls"}`)
	results := []HookResult{{Output: "not json"}}
	out := applyPreToolUseResults("bash", raw, results)
	if string(out) != string(raw) {
		t.Fatalf("got %s", string(out))
	}
}

func TestApplyPreToolUseResults_usesFirstValid(t *testing.T) {
	raw := []byte(`{"command":"old"}`)
	results := []HookResult{
		{Output: "nope"},
		{Output: `{"command":"new"}`},
	}
	out := applyPreToolUseResults("bash", raw, results)
	if string(out) != `{"command":"new"}` {
		t.Fatalf("got %s", string(out))
	}
}
