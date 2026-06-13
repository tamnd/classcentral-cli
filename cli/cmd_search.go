package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) searchCmd() *cobra.Command {
	var (
		subject  string
		provider string
		page     int
	)
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search online courses on Class Central",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(20)
			a.progressf("searching for %q...", args[0])
			courses, err := a.client.Search(cmd.Context(), args[0], page, n)
			if err != nil {
				return mapFetchErr(err)
			}
			// client-side subject / provider filters
			if subject != "" || provider != "" {
				filtered := courses[:0]
				for _, c := range courses {
					if subject != "" && !strings.Contains(strings.ToLower(c.Name), strings.ToLower(subject)) {
						continue
					}
					if provider != "" && !strings.EqualFold(c.Provider, provider) {
						continue
					}
					filtered = append(filtered, c)
				}
				courses = filtered
			}
			// re-rank after filter
			for i := range courses {
				courses[i].Rank = i + 1
			}
			return a.renderOrEmpty(courses, len(courses))
		},
	}
	cmd.Flags().StringVar(&subject, "subject", "", "filter results by subject keyword")
	cmd.Flags().StringVar(&provider, "provider", "", "filter results by provider name (e.g. Coursera)")
	cmd.Flags().IntVar(&page, "page", 1, "starting search page (1-based)")
	return cmd
}
