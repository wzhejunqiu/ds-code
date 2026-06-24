package rgutil

// IgnoreFlags returns ripgrep args for gitignore and hidden-file behavior.
func IgnoreFlags(respectGitignore, includeHidden bool) []string {
	var args []string
	if !respectGitignore {
		args = append(args, "--no-ignore", "--no-ignore-vcs", "--no-ignore-global")
	}
	if includeHidden {
		args = append(args, "--hidden")
	}
	return args
}
