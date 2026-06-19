package clipboard

import (
	"encoding/base64"
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
