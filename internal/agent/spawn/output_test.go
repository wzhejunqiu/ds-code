package spawn

import (
	"os"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

func testCfg(maxChars int) *config.Config {
	return &config.Config{
		Tools: config.ToolsConfig{
			Agent: config.AgentToolConfig{SummaryMaxChars: maxChars},
		},
	}
}

func TestShouldSpillResult_chars(t *testing.T) {
	max := 2048
	short := strings.Repeat("a", max)
	if ShouldSpillResult(short, max) {
		t.Fatal("at limit should inline")
	}
	if !ShouldSpillResult(short+"x", max) {
		t.Fatal("over char limit should spill")
	}
}

func TestShouldSpillResult_respectsConfigDefault(t *testing.T) {
	summary := strings.Repeat("a", 3000)
	if ShouldSpillResult(summary, 16000) {
		t.Fatal("3000 runes should inline when max is 16000")
	}
	if !ShouldSpillResult(summary, 2048) {
		t.Fatal("3000 runes should spill when max is 2048")
	}
}

func TestShouldSpillResult_bytes(t *testing.T) {
	s := strings.Repeat("a", 100)
	if ShouldSpillResult(s, 16000) {
		t.Fatal("short ascii should inline")
	}
}

func TestShouldSpillResult_byteCeiling(t *testing.T) {
	// Under rune limit but over 1 MiB byte cap must spill.
	s := strings.Repeat("a", InlineResultMaxBytes+1)
	if !ShouldSpillResult(s, 1_000_000) {
		t.Fatal("over byte ceiling should spill regardless of char limit")
	}
}

func TestDeliverResult_inline(t *testing.T) {
	cfg := testCfg(16000)
	d := DeliverResult(t.TempDir(), "sess", "tc", "hello", ResultCompleted, nil, cfg)
	if !d.Inline || d.Body != "hello" || d.OutputPath != "" {
		t.Fatalf("unexpected inline delivery: %+v", d)
	}
}

func TestDeliverResult_spill(t *testing.T) {
	dir := t.TempDir()
	cfg := testCfg(2048)
	long := strings.Repeat("x", 2049)
	d := DeliverResult(dir, "sess", "tc", long, ResultCompleted, nil, cfg)
	if d.Inline || d.OutputPath == "" {
		t.Fatalf("expected spill, got %+v", d)
	}
	data, err := os.ReadFile(d.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), long) {
		t.Fatal("file should contain full summary")
	}
}

func TestDeliverResult_inlineWithinDefaultLimit(t *testing.T) {
	dir := t.TempDir()
	cfg := testCfg(16000)
	summary := strings.Repeat("y", 3000)
	d := DeliverResult(dir, "sess", "tc", summary, ResultCompleted, nil, cfg)
	if !d.Inline || d.Body != summary {
		t.Fatalf("expected inline for 3000 runes with max 16000, got %+v", d)
	}
}

func TestXmlEscapeText(t *testing.T) {
	got := xmlEscapeText("a < b & c > d")
	want := "a &lt; b &amp; c &gt; d"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
