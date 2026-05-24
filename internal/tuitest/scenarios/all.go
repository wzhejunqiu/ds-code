//go:build tuitest

package scenarios

import (
	"encoding/json"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/llm"
)

// All returns built-in scenarios in stable order.
func All() []*Scenario {
	return []*Scenario{
		streamBasic(),
		streamReasoning(),
		toolRead(),
		toolGrep(),
		toolPatchSingle(),
		toolPatchMulti(),
		toolMulti(),
		toolShell(),
		errorAPI(),
		errorContext(),
		mdRich(),
		longStream(),
	}
}

// streamDemoDelay spaces SSE chunks so the harness TUI can repaint between deltas.
const streamDemoDelay = 120 * time.Millisecond

func reasoningParts(parts ...string) []StreamChunk {
	chunks := make([]StreamChunk, len(parts))
	for i, p := range parts {
		chunks[i] = StreamChunk{Reasoning: p}
		if i > 0 {
			chunks[i].Delay = streamDemoDelay
		}
	}
	return chunks
}

func contentParts(parts ...string) []StreamChunk {
	chunks := make([]StreamChunk, len(parts))
	for i, p := range parts {
		chunks[i] = StreamChunk{Content: p}
		if i > 0 {
			chunks[i].Delay = streamDemoDelay
		}
	}
	return chunks
}

func streamBasic() *Scenario {
	return &Scenario{
		Name:   "stream-basic",
		Prompt: "stream basic test",
		Turns: []Turn{{
			Chunks: []StreamChunk{
				{Content: "hel"},
				{Content: "lo ", Delay: streamDemoDelay},
				{Content: "world", Delay: streamDemoDelay},
			},
			FinishReason: "stop",
		}},
	}
}

func streamReasoning() *Scenario {
	chunks := reasoningParts(
		"Let ", "me ", "walk ", "through ", "this ", "step ", "by ", "step.\n\n",
		"First, ", "check ", "the ", "▾ ", "label ", "stays ", "open.\n",
		"Next, ", "watch ", "tiny ", "reasoning ", "chunks ", "stream ", "in.\n",
		"Then ", "content ", "starts; ", "thinking ", "collapses.\n",
	)
	chunks = append(chunks, contentParts("ans", "wer")...)
	return &Scenario{
		Name:   "stream-reasoning",
		Prompt: "reasoning stream test",
		Turns: []Turn{{
			Chunks:       chunks,
			FinishReason: "stop",
		}},
	}
}

func toolRead() *Scenario {
	args, _ := json.Marshal(map[string]string{"path": "sample.go"})
	return &Scenario{
		Name:   "tool-read",
		Prompt: "read sample file",
		Turns: []Turn{
			{
				ToolCalls: []llm.ToolCall{{
					ID: "call_read_1", Name: "read_file", Arguments: string(args),
				}},
				FinishReason: "tool_calls",
			},
			{
				Chunks:       []StreamChunk{{Content: "File read complete."}},
				FinishReason: "stop",
			},
		},
	}
}

func toolGrep() *Scenario {
	args, _ := json.Marshal(map[string]string{"pattern": "package", "path": "."})
	return &Scenario{
		Name:   "tool-grep",
		Prompt: "grep package",
		Turns: []Turn{
			{
				ToolCalls: []llm.ToolCall{{
					ID: "call_grep_1", Name: "grep", Arguments: string(args),
				}},
				FinishReason: "tool_calls",
			},
			{
				Chunks:       []StreamChunk{{Content: "Grep done."}},
				FinishReason: "stop",
			},
		},
	}
}

func toolPatchSingle() *Scenario {
	return toolPatchScenario(
		"tool-patch-single",
		"apply patch single",
		"call_patch_single",
		`*** Begin Patch
*** Update File: sample.go
@@
-func Hello() string { return "hello" }
+func Hello() string { return "hello" }
+// harness
*** End Patch`,
	)
}

func toolPatchMulti() *Scenario {
	return toolPatchScenario(
		"tool-patch-multi",
		"apply patch multi",
		"call_patch_multi",
		`*** Begin Patch
*** Update File: sample_multiline.go
@@
 func Hello() string {
 	return "hello"
 }
+// harness
*** End Patch`,
	)
}

func toolPatchScenario(name, prompt, callID, patch string) *Scenario {
	args, _ := json.Marshal(map[string]string{"patch": patch})
	return &Scenario{
		Name:   name,
		Prompt: prompt,
		Turns: []Turn{
			{
				ToolCalls: []llm.ToolCall{{
					ID: callID, Name: "apply_patch", Arguments: string(args),
				}},
				FinishReason: "tool_calls",
			},
			{
				Chunks:       []StreamChunk{{Content: "Patch applied."}},
				FinishReason: "stop",
			},
		},
	}
}

func toolMulti() *Scenario {
	rargs, _ := json.Marshal(map[string]string{"path": "sample.go"})
	gargs, _ := json.Marshal(map[string]string{"pattern": "func", "path": "sample.go"})
	return &Scenario{
		Name:   "tool-multi",
		Prompt: "multi tools",
		Turns: []Turn{
			{
				ToolCalls: []llm.ToolCall{
					{ID: "call_r1", Name: "read_file", Arguments: string(rargs)},
					{ID: "call_g1", Name: "grep", Arguments: string(gargs)},
				},
				FinishReason: "tool_calls",
			},
			{
				Chunks:       []StreamChunk{{Content: "Both tools finished."}},
				FinishReason: "stop",
			},
		},
	}
}

func toolShell() *Scenario {
	args, _ := json.Marshal(map[string]string{"command": "echo harness-ok"})
	return &Scenario{
		Name:   "tool-shell",
		Prompt: "run shell",
		Turns: []Turn{
			{
				ToolCalls: []llm.ToolCall{{
					ID: "call_sh_1", Name: "shell", Arguments: string(args),
				}},
				FinishReason: "tool_calls",
			},
			{
				Chunks:       []StreamChunk{{Content: "Shell done."}},
				FinishReason: "stop",
			},
		},
	}
}

func errorAPI() *Scenario {
	return &Scenario{
		Name:   "error-api",
		Prompt: "trigger api error",
		Turns: []Turn{{
			HTTPStatus: 400,
			ErrBody:    `{"error":{"message":"bad request","type":"invalid_request"}}`,
		}},
	}
}

func errorContext() *Scenario {
	return &Scenario{
		Name:   "error-context",
		Prompt: "context compact retry",
		Turns: []Turn{
			{
				Chunks:       []StreamChunk{{Content: "after compact"}},
				FinishReason: "stop",
			},
		},
	}
}

func mdRich() *Scenario {
	md := "# Title\n\n**bold** and `code`.\n\n```go\nfmt.Println(\"hi\")\n```\n"
	return &Scenario{
		Name:   "md-rich",
		Prompt: "markdown render",
		Turns: []Turn{{
			Chunks:       []StreamChunk{{Content: md}},
			FinishReason: "stop",
		}},
	}
}

func longStream() *Scenario {
	chunks := make([]StreamChunk, 0, 40)
	for i := 0; i < 40; i++ {
		chunks = append(chunks, StreamChunk{Content: "x", Delay: 2 * time.Millisecond})
	}
	return &Scenario{
		Name:   "long-stream",
		Prompt: "long stream",
		Turns: []Turn{{
			Chunks:       chunks,
			FinishReason: "stop",
		}},
	}
}
