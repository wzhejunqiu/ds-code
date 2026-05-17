package commands

import (
	"github.com/hejunqiu/ds-code/cmd/ds-code/app"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/spf13/cobra"
)

// ResumeCmd resumes an interactive TUI session by ID.
func ResumeCmd() *cobra.Command {
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
			closeLog, err := logging.Setup(logging.Options{
				ProjectRoot: cfg.ProjectRoot,
				Verbosity:   cfg.LogVerbosity,
			})
			if err != nil {
				return err
			}
			defer closeLog()

			return app.New(cfg).RunTUI(cmd, args[0])
		},
	}
}
