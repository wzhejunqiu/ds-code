package context

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

const gitStatusMaxChars = 2048

const gitStatusSectionTruncated = "\n...[status 已截断]"

// CaptureGitSnapshot captures structured git context for the system prompt.
// Returns empty string when not a git repository.
func CaptureGitSnapshot(projectRoot string, maxChars int) (string, error) {
	if maxChars <= 0 {
		maxChars = 16000
	}
	if !isGitRepo(projectRoot) {
		return "", nil
	}

	var b strings.Builder

	currentBranch := "(none)"
	if out, err := runGit(projectRoot, "branch", "--show-current"); err == nil {
		if s := strings.TrimSpace(out); s != "" {
			currentBranch = s
		}
	}
	appendGitSection(&b, "当前分支", currentBranch)

	appendGitSection(&b, "默认分支", defaultBranchName(projectRoot))

	gitUser := formatGitUser(projectRoot)
	appendGitSection(&b, "Git 用户", gitUser)

	statusBody := "(unavailable)"
	if out, err := runGit(projectRoot, "status"); err == nil {
		statusBody = truncateGitSection(strings.TrimSpace(out), gitStatusMaxChars)
	}
	appendGitSection(&b, "git status", statusBody)

	recentCommits := "(no commits)"
	if out, err := runGit(projectRoot, "log", "-5", "--format=%h %s (%an, %ar)"); err == nil {
		if s := strings.TrimSpace(out); s != "" {
			recentCommits = s
		}
	}
	appendGitSection(&b, "最近提交", recentCommits)

	s := strings.TrimSpace(b.String())
	if len(s) > maxChars {
		s = s[:maxChars] + GitSnapshotTruncated
	}
	return s, nil
}

func defaultBranchName(dir string) string {
	if out, err := runGit(dir, "config", "--get", "init.defaultBranch"); err == nil {
		if s := strings.TrimSpace(out); s != "" {
			return s
		}
	}
	if out, err := runGit(dir, "symbolic-ref", "refs/remotes/origin/HEAD"); err == nil {
		ref := strings.TrimSpace(out)
		const prefix = "refs/remotes/origin/"
		if strings.HasPrefix(ref, prefix) {
			if name := strings.TrimPrefix(ref, prefix); name != "" {
				return name
			}
		}
	}
	return "main"
}

func formatGitUser(dir string) string {
	name, _ := runGit(dir, "config", "user.name")
	email, _ := runGit(dir, "config", "user.email")
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	switch {
	case name != "" && email != "":
		return name + " <" + email + ">"
	case name != "":
		return name
	case email != "":
		return email
	default:
		return "(not configured)"
	}
}

func appendGitSection(b *strings.Builder, title, body string) {
	b.WriteString("### ")
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(strings.TrimSpace(body))
	b.WriteString("\n\n")
}

func truncateGitSection(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + gitStatusSectionTruncated
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
