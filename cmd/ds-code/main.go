package main

import (
	"fmt"
	"os"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/spf13/cobra"
)

var version = "0.1.0-dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ds-code",
		Short: "DeepSeek-native coding agent CLI",
		Long:  "ds-code: Agent/Plan modes, Codex-style apply_patch, and project context (AGENTS.md, rules, skills).",
	}

	config.BindFlags(root)

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	})

	root.AddCommand(sessionsCmd())
	root.AddCommand(resumeCmd())

	root.RunE = runRoot
	return root
}

func runRoot(cmd *cobra.Command, _ []string) error {
	requireKey := false
	prompt, _ := cmd.Flags().GetString("prompt")
	if prompt != "" {
		requireKey = true
	}

	cfg, err := config.Load(cmd, config.Options{RequireAPIKey: requireKey})
	if err != nil {
		return err
	}
	if err := config.ApplyCLIDerived(cfg, cmd); err != nil {
		return err
	}

	application := &app{cfg: cfg}

	if cfg.Prompt != "" {
		return application.runNonInteractive(cmd)
	}

	if permissionIsTTY() {
		return application.runREPL(cmd)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "stdin is not a TTY. Use: ds-code -p \"your task\"")
	return nil
}

func permissionIsTTY() bool {
	// local wrapper to avoid importing permission in main for tiny helper
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func sessionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sessions",
		Short: "List saved sessions (Phase 3+)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(cmd, config.Options{})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Sessions DB (Phase 3): %s\n", config.DefaultDBPath(cfg.ProjectRoot))
			return nil
		},
	}
}

func resumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume [session_id]",
		Short: "Resume a session (Phase 3+)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.Load(cmd, config.Options{RequireAPIKey: true})
			if err != nil {
				return err
			}
			return fmt.Errorf("resume %q: not implemented (Phase 3)", args[0])
		},
	}
}
