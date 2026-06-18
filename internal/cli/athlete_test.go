package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jiahongchen/race-results/internal/domain"
	"github.com/jiahongchen/race-results/internal/provider"
)

type stubAthlete struct {
	many bool
}

func (stubAthlete) Name() string { return "athlinks" }
func (stubAthlete) Lookup(_ context.Context, _ domain.Event, _ string) (domain.Result, error) {
	return domain.Result{}, nil
}
func (s stubAthlete) FindAthletes(_ context.Context, _ string) ([]domain.Athlete, error) {
	if s.many {
		return []domain.Athlete{{ID: "1", Name: "Jane A"}, {ID: "2", Name: "Jane B"}}, nil
	}
	return []domain.Athlete{{ID: "1", Name: "Jane A"}}, nil
}
func (stubAthlete) AthleteHistory(_ context.Context, id string) ([]domain.Result, error) {
	return []domain.Result{{Provider: "athlinks", RaceName: "Berlin", Date: "2024-09-29", NetTime: "02:45:10"}}, nil
}

func TestAthleteSingleMatch(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(stubAthlete{many: false})
	root := NewRoot(reg)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"athlete", "Jane A"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "Berlin") {
		t.Fatalf("expected history:\n%s", out.String())
	}
}

func TestAthleteManyMatches(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(stubAthlete{many: true})
	root := NewRoot(reg)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"athlete", "Jane"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "Jane A") || !strings.Contains(out.String(), "Jane B") {
		t.Fatalf("expected athlete list to refine:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "--racer-id") {
		t.Fatalf("expected refine hint:\n%s", out.String())
	}
}
