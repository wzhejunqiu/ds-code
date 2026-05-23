package main

import (
	"fmt"
	"os"

	"github.com/hejunqiu/ds-code/cmd/ds-code/app"
	"github.com/hejunqiu/ds-code/cmd/ds-code/commands"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/version"
	"github.com/hejunqiu/ds-code/internal/versioninfo"
	"github.com/spf13/cobra"
)

var gitCommit string // set at release build via -ldflags; empty uses vcs info from go build

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
			versioninfo.Write(cmd.OutOrStdout(), version.Version, gitCommit)
		},
	})

	root.AddCommand(commands.SessionsCmd())
	root.AddCommand(commands.ResumeCmd())

	root.RunE = runRoot
	return root
}

func runRoot(cmd *cobra.Command, _ []string) error {
	prompt, _ := cmd.Flags().GetString("prompt")
	// Interactive TUI and -p both call the LLM; only the non-TTY idle path skips it.
	requireKey := prompt != "" || permission.IsInteractiveTTY()

	cfg, err := config.Load(cmd, config.Options{RequireAPIKey: requireKey})
	if err != nil {
		return err
	}
	if err := config.ApplyCLIDerived(cfg, cmd); err != nil {
		return err
	}
	closeLog, err := setupLogging(cfg)
	if err != nil {
		return err
	}
	defer closeLog()
	app.LogConfigResolved(cfg)
	app.MaybeWarnSensitiveLog(cfg, os.Stderr)

	application := app.New(cfg)

	if cfg.Prompt != "" {
		return application.RunNonInteractive(cmd)
	}

	if permission.IsInteractiveTTY() {
		return application.RunTUI(cmd, "")
	}

	fmt.Fprintln(cmd.OutOrStdout(), "stdin is not a TTY. Use: ds-code -p \"your task\"")
	return nil
}

func setupLogging(cfg *config.Config) (func(), error) {
	return logging.Setup(logging.Options{
		ProjectRoot:        cfg.ProjectRoot,
		Verbosity:          cfg.LogVerbosity,
		AllowSensitiveData: cfg.AllowLogSensitiveData,
	})
}
