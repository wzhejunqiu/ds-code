package clipboard

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Write copies text to the system clipboard.
func Write(text string) error {
	if text == "" {
		return nil
	}
	if err := writePlatform(text); err == nil {
		return nil
	}
	if err := writeOSC52(text); err == nil {
		return nil
	}
	return fmt.Errorf("clipboard: no backend available")
}

func writePlatform(text string) error {
	switch runtime.GOOS {
	case "darwin":
		return runClipboard("pbcopy", text)
	case "linux":
		if err := runClipboard("wl-copy", text); err == nil {
			return nil
		}
		if err := runClipboard("xclip", "-selection", "clipboard", text); err == nil {
			return nil
		}
		return runClipboard("xsel", "--clipboard", "--input", text)
	case "windows":
		return runClipboard("clip.exe", text)
	default:
		return fmt.Errorf("clipboard: unsupported platform %s", runtime.GOOS)
	}
}

func encodeOSC52(text string) string {
	return "\033]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\007"
}

func writeOSC52(text string) error {
	_, err := fmt.Fprint(os.Stdout, encodeOSC52(text))
	return err
}

func runClipboard(name string, args ...string) error {
	var stdin string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		stdin = args[0]
		args = args[1:]
	}
	cmd := exec.Command(name, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clipboard: %s: %w", name, err)
	}
	return nil
}
