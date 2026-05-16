package permission

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Prompter asks the user to approve a write/shell operation.
type Prompter func(tool, summary string) (allowed bool, err error)

// StdinPrompter prompts on out (typically stderr) and reads from stdin.
func StdinPrompter(out io.Writer) Prompter {
	return func(tool, summary string) (bool, error) {
		if summary == "" {
			summary = "(no details)"
		}
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		fmt.Fprintf(out, "\n[permission] Allow %s?\n  %s\n[y/N]: ", tool, summary)
		sc := bufio.NewScanner(os.Stdin)
		if !sc.Scan() {
			return false, sc.Err()
		}
		ans := strings.TrimSpace(strings.ToLower(sc.Text()))
		return ans == "y" || ans == "yes", nil
	}
}
