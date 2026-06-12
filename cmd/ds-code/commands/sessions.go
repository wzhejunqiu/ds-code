package commands

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/wzhejunqiu/ds-code/cmd/ds-code/app"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	sessionsqlite "github.com/wzhejunqiu/ds-code/internal/session/sqlite"
)

// SessionsCmd lists saved sessions for the current project.
func SessionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sessions",
		Short: "List saved sessions for the current project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := app.BootstrapConfig(cmd, config.Options{})
			if err != nil {
				return err
			}
			defer logging.TrySetup(logging.Options{
				ProjectRoot:        cfg.ProjectRoot,
				Verbosity:          cfg.LogVerbosity,
				AllowSensitiveData: cfg.AllowLogSensitiveData,
			})()

			store, err := sessionsqlite.OpenDefault(cfg.ProjectRoot)
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
