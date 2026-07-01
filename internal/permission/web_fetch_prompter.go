package permission

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// WebFetchPromptRequest is sent to the TUI for out-of-allowlist web_fetch hosts.
type WebFetchPromptRequest struct {
	Host  string
	URL   string
	Reply chan WebFetchChoice
}

// TUIWebFetchPrompter blocks until the TUI responds with a WebFetchChoice.
func TUIWebFetchPrompter(reqCh chan<- WebFetchPromptRequest) WebFetchPrompter {
	return func(host, rawURL string) (WebFetchChoice, error) {
		reply := make(chan WebFetchChoice, 1)
		reqCh <- WebFetchPromptRequest{Host: host, URL: rawURL, Reply: reply}
		return <-reply, nil
	}
}

// StdinWebFetchPrompter prompts on out for non-TUI interactive sessions.
func StdinWebFetchPrompter(out io.Writer) WebFetchPrompter {
	return func(host, rawURL string) (WebFetchChoice, error) {
		if rawURL != "" {
			fmt.Fprintf(out, "\n[permission] web_fetch: %s 不在 allowlist\n  %s\n", host, rawURL)
		} else {
			fmt.Fprintf(out, "\n[permission] web_fetch: %s 不在 allowlist\n", host)
		}
		fmt.Fprint(out, "  [1/a] 允许本次  [2/s] 始终允许  [3/d] 拒绝\n选择: ")
		sc := bufio.NewScanner(os.Stdin)
		if !sc.Scan() {
			return WebFetchDeny, sc.Err()
		}
		switch strings.ToLower(strings.TrimSpace(sc.Text())) {
		case "1", "a", "allow", "y", "yes":
			return WebFetchAllowOnce, nil
		case "2", "s", "always", "save":
			return WebFetchAllowAlways, nil
		case "3", "d", "deny", "n", "no":
			return WebFetchDeny, nil
		default:
			return WebFetchDeny, nil
		}
	}
}
