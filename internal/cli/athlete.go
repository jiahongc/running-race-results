package cli

import (
	"fmt"
	"os"

	"github.com/jiahongchen/race-results/internal/domain"
	"github.com/jiahongchen/race-results/internal/provider"
	"github.com/jiahongchen/race-results/internal/provider/athlinks"
	"github.com/jiahongchen/race-results/internal/render"
	"github.com/spf13/cobra"
)

var historyCols = []render.Column{
	{Header: "Date", Value: func(r domain.Result) string { return r.Date }},
	{Header: "Race", Value: func(r domain.Result) string { return r.RaceName }},
	{Header: "Distance", Value: func(r domain.Result) string { return r.Distance }},
	{Header: "Net time", Value: func(r domain.Result) string { return r.NetTime }},
	{Header: "Overall", Value: func(r domain.Result) string {
		if r.OverallPlace > 0 {
			return fmt.Sprintf("%d", r.OverallPlace)
		}
		return ""
	}},
}

func newAthleteCmd(reg *provider.Registry) *cobra.Command {
	var racerID string
	var useMe bool
	var asJSON bool

	cmd := &cobra.Command{
		Use:   `athlete "<name>"`,
		Short: "Show a runner's race history across events (Athlinks)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			as := firstAthleteSearcher(reg)
			if as == nil {
				return fmt.Errorf("no athlete-search provider registered")
			}

			// Resolve a racer id: --me, --racer-id, or a name search.
			id := racerID
			if useMe {
				tok := getenvAthlinksToken()
				if tok == "" {
					return fmt.Errorf("athlete --me: ATHLINKS_TOKEN not set")
				}
				rid, err := athlinks.RacerIDFromToken(tok)
				if err != nil {
					return err
				}
				id = rid
			}
			if id == "" {
				if len(args) == 0 {
					return fmt.Errorf("provide a name, or --racer-id, or --me")
				}
				found, err := as.FindAthletes(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				switch len(found) {
				case 0:
					return fmt.Errorf("no athlete matching %q", args[0])
				case 1:
					id = found[0].ID
				default:
					fmt.Fprintln(cmd.OutOrStdout(), "Multiple athletes — refine with --racer-id <id>:")
					return render.Athletes(cmd.OutOrStdout(), found)
				}
			}

			history, err := as.AthleteHistory(cmd.Context(), id)
			if err != nil {
				return err
			}
			if asJSON {
				return render.JSONValue(cmd.OutOrStdout(), history)
			}
			if len(history) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no races found")
				return nil
			}
			return render.List(cmd.OutOrStdout(), history, historyCols)
		},
	}
	cmd.Flags().StringVar(&racerID, "racer-id", "", "Athlinks racer id (skip name search)")
	cmd.Flags().BoolVar(&useMe, "me", false, "use the racer id from ATHLINKS_TOKEN")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func firstAthleteSearcher(reg *provider.Registry) provider.AthleteSearcher {
	for _, name := range reg.Names() {
		if p, ok := reg.Get(name); ok {
			if as, ok := p.(provider.AthleteSearcher); ok {
				return as
			}
		}
	}
	return nil
}

func getenvAthlinksToken() string { return os.Getenv("ATHLINKS_TOKEN") }
