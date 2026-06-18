// internal/provider/raceresult/raceresult_test.go
package raceresult

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jiahongchen/race-results/internal/domain"
	"github.com/jiahongchen/race-results/internal/provider"
)

// loadFixture reads a fixture file, strips the top-level "_meta" key,
// and returns the resulting JSON bytes (the real API response).
func loadFixture(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("loadFixture: read %s: %v", path, err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("loadFixture: unmarshal %s: %v", path, err)
	}
	delete(m, "_meta")
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("loadFixture: re-marshal %s: %v", path, err)
	}
	return out
}

func TestLookup(t *testing.T) {
	configFixture := loadFixture(t, "../../../testdata/fixtures/raceresult/config.json")
	resultsFixture := loadFixture(t, "../../../testdata/fixtures/raceresult/results.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/results/config"):
			w.Write(configFixture)
		case strings.Contains(r.URL.Path, "/results/list"):
			w.Write(resultsFixture)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New()
	c.BaseURL = srv.URL
	c.DataBaseURL = srv.URL

	ev := domain.Event{
		Provider: "raceresult",
		ID:       "390537",
		Name:     "17. REWE Team Challenge Dresden",
		Year:     2026,
	}

	t.Run("hit", func(t *testing.T) {
		result, err := c.Lookup(context.Background(), ev, "1286")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Runner != "Theresa Menzel" {
			t.Errorf("Runner: got %q, want %q", result.Runner, "Theresa Menzel")
		}
		if result.Bib != "1286" {
			t.Errorf("Bib: got %q, want %q", result.Bib, "1286")
		}
		if result.NetTime != "0:17:43" {
			t.Errorf("NetTime: got %q, want %q", result.NetTime, "0:17:43")
		}
		if result.OverallPlace != 1 {
			t.Errorf("OverallPlace: got %d, want %d", result.OverallPlace, 1)
		}
		if result.Provider != "raceresult" {
			t.Errorf("Provider: got %q, want %q", result.Provider, "raceresult")
		}
	})

	t.Run("miss", func(t *testing.T) {
		_, err := c.Lookup(context.Background(), ev, "000000")
		if !errors.Is(err, provider.ErrBibNotFound) {
			t.Errorf("expected ErrBibNotFound, got %v", err)
		}
	})
}
