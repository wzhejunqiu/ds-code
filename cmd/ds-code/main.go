package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/spf13/cobra"
)

var version = "0.0.0-dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
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

	if cfg.Prompt != "" {
		return runNonInteractive(cmd, cfg)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Interactive TUI is planned for Phase 4.")
	fmt.Fprintln(cmd.OutOrStdout(), "Use: ds-code -p \"your task\"  (Phase 1+ Agent MVP)")
	fmt.Fprintln(cmd.OutOrStdout(), "Config loaded — project:", cfg.ProjectRoot)
	return nil
}

func runNonInteractive(cmd *cobra.Command, cfg *config.Config) error {
	msg := map[string]any{
		"status":  "not_implemented",
		"phase":   "Phase 1",
		"message": "Non-interactive Agent (-p) will be available in Phase 1.",
		"prompt":  cfg.Prompt,
	}
	if cfg.JSONOutput {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(msg)
	}
	fmt.Fprintln(cmd.OutOrStdout(), msg["message"])
	return nil
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
