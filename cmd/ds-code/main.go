package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/session"
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
		return application.runTUI(cmd, "")
	}

	fmt.Fprintln(cmd.OutOrStdout(), "stdin is not a TTY. Use: ds-code -p \"your task\"")
	return nil
}

func permissionIsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func sessionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sessions",
		Short: "List saved sessions for the current project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(cmd, config.Options{})
			if err != nil {
				return err
			}
			store, err := session.OpenDefaultStore(cfg.ProjectRoot)
			if err != nil {
				return err
			}
			defer store.Close()

			list, err := store.ListSessions(context.Background(), 50)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(list) == 0 {
				fmt.Fprintf(out, "No sessions in %s\n", config.DefaultDBPath(cfg.ProjectRoot))
				return nil
			}
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tMODEL\tTOKENS\tUPDATED")
			for _, s := range list {
				title := s.Title
				if len(title) > 40 {
					title = title[:37] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
					s.ID, title, s.Model, s.BilledTokens, s.UpdatedAt.Format(time.RFC3339))
			}
			return w.Flush()
		},
	}
}

func resumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <session_id>",
		Short: "Resume an interactive session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cmd, config.Options{RequireAPIKey: true})
			if err != nil {
				return err
			}
			if err := config.ApplyCLIDerived(cfg, cmd); err != nil {
				return err
			}
			application := &app{cfg: cfg}
			return application.runTUI(cmd, args[0])
		},
	}
}
