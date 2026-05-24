package app

import (
	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/spf13/cobra"
)

// BootstrapConfig loads config, applies user billing overrides, and merges CLI flags.
func BootstrapConfig(cmd *cobra.Command, opts config.Options) (*config.Config, error) {
	cfg, err := config.Load(cmd, opts)
	if err != nil {
		return nil, err
	}
	if err := billing.SetupFromUserConfig(); err != nil {
		return nil, err
	}
	if err := config.ApplyCLIDerived(cfg, cmd); err != nil {
		return nil, err
	}
	return cfg, nil
}
