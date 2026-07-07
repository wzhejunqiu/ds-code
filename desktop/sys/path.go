package sys

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DepStatus reports whether a dependency binary is available.
type DepStatus struct {
	Name  string `json:"name"`
	Found bool   `json:"found"`
	Path  string `json:"path,omitempty"`
	Hint  string `json:"hint,omitempty"`
}

var defaultDeps = []struct {
	name string
	hint string
}{
	{"git", "Install Xcode Command Line Tools: xcode-select --install"},
	{"node", "Install Node.js from https://nodejs.org or via Homebrew: brew install node"},
	{"gopls", "Install gopls: go install golang.org/x/tools/gopls@latest"},
}

// EnsurePATH prepends common macOS binary locations to PATH for subprocesses.
func EnsurePATH() {
	home, _ := os.UserHomeDir()
	candidates := []string{
		"/usr/bin",
		"/bin",
		"/usr/sbin",
		"/sbin",
		"/opt/homebrew/bin",
		"/usr/local/bin",
		filepath.Join(home, ".local", "bin"),
	}
	current := os.Getenv("PATH")
	parts := strings.Split(current, string(os.PathListSeparator))
	seen := make(map[string]bool)
	for _, p := range parts {
		if p != "" {
			seen[p] = true
		}
	}
	var prefix []string
	for _, c := range candidates {
		if !seen[c] {
			prefix = append(prefix, c)
			seen[c] = true
		}
	}
	if len(prefix) > 0 {
		_ = os.Setenv("PATH", strings.Join(append(prefix, parts...), string(os.PathListSeparator)))
	}
}

// CheckDependencies probes common tool dependencies on PATH.
func CheckDependencies() []DepStatus {
	out := make([]DepStatus, 0, len(defaultDeps))
	for _, d := range defaultDeps {
		st := DepStatus{Name: d.name, Hint: d.hint}
		if p, err := exec.LookPath(d.name); err == nil {
			st.Found = true
			st.Path = p
		}
		out = append(out, st)
	}
	return out
}

// ExtendedPATH returns the PATH after EnsurePATH normalization.
func ExtendedPATH() string {
	EnsurePATH()
	return os.Getenv("PATH")
}
