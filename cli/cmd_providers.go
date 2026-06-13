package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) providersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List all course providers indexed by Class Central",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a.progressf("fetching providers...")
			providers, err := a.client.Providers(cmd.Context())
			if err != nil {
				return mapFetchErr(err)
			}
			if n := a.limit; n > 0 && len(providers) > n {
				providers = providers[:n]
			}
			return a.renderOrEmpty(providers, len(providers))
		},
	}
}
