// internal/cli/lookup_test.go
package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jiahongchen/race-results/internal/domain"
	"github.com/jiahongchen/race-results/internal/provider"
)

type stubProvider struct{}

func (stubProvider) Name() string { return "mika" }
func (stubProvider) Lookup(_ context.Context, e domain.Event, bib string) (domain.Result, error) {
	return domain.Result{Provider: "mika", RaceName: e.Name, Year: e.Year, Bib: bib, Runner: "Jane Doe", NetTime: "02:45:10"}, nil
}

func TestLookupRendersResult(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(stubProvider{})
	root := NewRoot(reg)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"lookup", "berlin marathon", "1234", "--year", "2025"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "Jane Doe") {
		t.Fatalf("expected runner in output:\n%s", out.String())
	}
}

func TestLookupUnknownRaceErrors(t *testing.T) {
	reg := provider.NewRegistry()
	root := NewRoot(reg)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"lookup", "zzz nonexistent", "1"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown race")
	}
}
