package context

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// CaptureGitSnapshot runs git status and diff --stat in projectRoot.
// Returns empty string when not a git repository.
func CaptureGitSnapshot(projectRoot string, maxChars int) (string, error) {
	if maxChars <= 0 {
		maxChars = 16000
	}
	if !isGitRepo(projectRoot) {
		return "", nil
	}

	var b strings.Builder
	b.WriteString("### git status -sb\n")
	if out, err := runGit(projectRoot, "status", "-sb"); err == nil {
		b.WriteString(out)
	} else {
		b.WriteString("(unavailable)\n")
	}
	b.WriteString("\n### git diff --stat\n")
	if out, err := runGit(projectRoot, "diff", "--stat"); err == nil {
		b.WriteString(out)
	} else {
		b.WriteString("(unavailable)\n")
	}

	s := strings.TrimSpace(b.String())
	if len(s) > maxChars {
		s = s[:maxChars] + "\n... [git snapshot truncated]"
	}
	return s, nil
}

func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, stderr.String())
		}
		return "", err
	}
	return stdout.String(), nil
}
