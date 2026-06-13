package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) topCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "top",
		Short: "Top free online courses on Class Central",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(50)
			a.progressf("fetching top %d free courses...", n)
			courses, err := a.client.Top(cmd.Context(), n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(courses, len(courses))
		},
	}
}
