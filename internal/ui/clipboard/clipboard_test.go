package clipboard

import (
	"encoding/base64"
	"runtime"
	"strings"
	"testing"
)

func TestEncodeOSC52(t *testing.T) {
	got := encodeOSC52("hello")
	if !strings.HasPrefix(got, "\033]52;c;") || !strings.HasSuffix(got, "\007") {
		t.Fatalf("unexpected OSC52 frame: %q", got)
	}
	payload := strings.TrimPrefix(got, "\033]52;c;")
	payload = strings.TrimSuffix(payload, "\007")
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != "hello" {
		t.Fatalf("got %q", decoded)
	}
}

func TestClipboard_write_platform(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("platform clipboard test only on darwin/linux CI")
	}
	// Best-effort: ensure Write does not panic; may fail without display server.
	_ = Write("ds-code-test-clipboard")
}
