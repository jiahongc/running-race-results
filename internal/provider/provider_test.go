// internal/provider/provider_test.go
package provider

import (
	"context"
	"testing"

	"github.com/jiahongchen/race-results/internal/domain"
)

type fakeProvider struct{}

func (fakeProvider) Name() string { return "fake" }
func (fakeProvider) Lookup(_ context.Context, e domain.Event, bib string) (domain.Result, error) {
	if bib == "0" {
		return domain.Result{}, ErrBibNotFound
	}
	return domain.Result{Provider: "fake", Bib: bib, Runner: "Test Runner", RaceName: e.Name}, nil
}

func TestRegistryGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(fakeProvider{})
	p, ok := reg.Get("fake")
	if !ok {
		t.Fatal("expected provider registered")
	}
	res, err := p.Lookup(context.Background(), domain.Event{Name: "Test Race"}, "12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Bib != "12" || res.RaceName != "Test Race" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestRegistryMissing(t *testing.T) {
	reg := NewRegistry()
	if _, ok := reg.Get("nope"); ok {
		t.Fatal("expected missing provider")
	}
}
