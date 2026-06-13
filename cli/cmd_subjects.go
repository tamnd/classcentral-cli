package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) subjectsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "subjects",
		Short: "List all subjects in the Class Central catalog",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a.progressf("fetching subjects...")
			subjects, err := a.client.Subjects(cmd.Context())
			if err != nil {
				return mapFetchErr(err)
			}
			if n := a.limit; n > 0 && len(subjects) > n {
				subjects = subjects[:n]
			}
			return a.renderOrEmpty(subjects, len(subjects))
		},
	}
}
