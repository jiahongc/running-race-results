// internal/cli/lookup.go
package cli

import (
	"fmt"
	"time"

	"github.com/jiahongchen/race-results/internal/catalog"
	"github.com/jiahongchen/race-results/internal/provider"
	"github.com/jiahongchen/race-results/internal/render"
	"github.com/jiahongchen/race-results/internal/resolve"
	"github.com/spf13/cobra"
)

func newLookupCmd(reg *provider.Registry) *cobra.Command {
	var year int
	var date string
	var asJSON bool

	cmd := &cobra.Command{
		Use:   `lookup "<race name>" <bib>`,
		Short: "Resolve a race and return the result for a bib",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			race, bib := args[0], args[1]

			if year == 0 && date != "" {
				t, perr := time.Parse("2006-01-02", date)
				if perr != nil {
					return fmt.Errorf("invalid --date %q (want YYYY-MM-DD): %w", date, perr)
				}
				year = t.Year()
			}

			entries, err := catalog.Load()
			if err != nil {
				return err
			}
			cands, err := resolve.Resolve(entries, race, year)
			if err != nil {
				return fmt.Errorf("%w: %q", err, race)
			}
			// Ambiguity: two candidates within 0.05 of each other.
			if len(cands) > 1 && cands[0].Score-cands[1].Score < 0.05 {
				if asJSON {
					return fmt.Errorf("ambiguous race %q; candidates: %s, %s",
						race, cands[0].Event.Name, cands[1].Event.Name)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Multiple matches — refine with --year or a fuller name:")
				for _, c := range cands {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%d)\n", c.Event.Name, c.Event.Year)
				}
				return nil
			}

			ev := cands[0].Event
			p, ok := reg.Get(ev.Provider)
			if !ok {
				return fmt.Errorf("no adapter registered for provider %q", ev.Provider)
			}
			res, err := p.Lookup(cmd.Context(), ev, bib)
			if err != nil {
				return err
			}
			if asJSON {
				return render.JSON(cmd.OutOrStdout(), res)
			}
			return render.Table(cmd.OutOrStdout(), res)
		},
	}
	cmd.Flags().IntVar(&year, "year", 0, "race edition year")
	cmd.Flags().StringVar(&date, "date", "", "race date YYYY-MM-DD (year is derived if --year unset)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}
