// internal/render/render.go
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/jiahongchen/race-results/internal/domain"
)

// Table writes a human-readable two-column view; empty fields are skipped.
func Table(w io.Writer, r domain.Result) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	row := func(label, val string) {
		if val != "" {
			fmt.Fprintf(tw, "%s\t%s\n", label, val)
		}
	}
	row("Race", fmt.Sprintf("%s %d", r.RaceName, r.Year))
	row("Runner", r.Runner)
	row("Bib", r.Bib)
	row("Net time", r.NetTime)
	row("Gun time", r.GunTime)
	if r.OverallPlace > 0 {
		row("Overall place", fmt.Sprintf("%d", r.OverallPlace))
	}
	row("Age group", r.AgeGroup)
	row("Source", r.SourceURL)
	return tw.Flush()
}

// JSON writes the result as indented JSON.
func JSON(w io.Writer, r domain.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
