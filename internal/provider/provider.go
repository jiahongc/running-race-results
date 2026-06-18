// internal/provider/provider.go
package provider

import (
	"context"
	"errors"

	"github.com/jiahongchen/race-results/internal/domain"
)

// ErrBibNotFound means the event resolved but no runner has that bib.
var ErrBibNotFound = errors.New("bib not found")

// Provider looks up a single runner by bib within a resolved event.
type Provider interface {
	Name() string
	Lookup(ctx context.Context, event domain.Event, bib string) (domain.Result, error)
}

// Registry maps provider names to implementations.
type Registry struct {
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

func (r *Registry) Register(p Provider) {
	r.providers[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}
