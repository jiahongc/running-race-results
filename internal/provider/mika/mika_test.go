// internal/provider/mika/mika_test.go
package mika

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/jiahongchen/race-results/internal/domain"
	"github.com/jiahongchen/race-results/internal/provider"
)

// reHeaderLine strips a leading "### ...\n" line (printing-press capture format).
var reHeaderLine = regexp.MustCompile(`(?m)^###[^\n]*\n`)

// loadFixtureHTML reads a printing-press HTML fixture (stored as a JSON-encoded
// string, optionally preceded by a "### Result" header line) and returns the
// raw HTML bytes ready to serve.
func loadFixtureHTML(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	// Strip optional leading header line(s).
	cleaned := reHeaderLine.ReplaceAll(raw, nil)
	cleaned = []byte(strings.TrimSpace(string(cleaned)))

	// The file may contain extra content after the closing JSON quote
	// (e.g. a trailing "### Ran Playwright code" block).  Find the end of
	// the first JSON string value (opening " at index 0).
	if len(cleaned) == 0 || cleaned[0] != '"' {
		t.Fatalf("fixture %s: expected JSON string, got %q…", path, string(cleaned[:min(20, len(cleaned))]))
	}
	end := 1
	for end < len(cleaned) {
		if cleaned[end] == '"' && cleaned[end-1] != '\\' {
			break
		}
		end++
	}
	jsonStr := cleaned[:end+1]

	var decoded string
	if err := json.Unmarshal(jsonStr, &decoded); err != nil {
		t.Fatalf("fixture %s: json.Unmarshal: %v", path, err)
	}
	return []byte(decoded)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	searchHTML := loadFixtureHTML(t, "../../../testdata/fixtures/mika/search.html")
	detailHTML := loadFixtureHTML(t, "../../../testdata/fixtures/mika/detail.html")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("content") == "detail" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(detailHTML)
			return
		}
		// Default: search page (pid=search).
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(searchHTML)
	}))
	return srv
}

func TestLookup_Hit(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	c := New()
	c.BaseURL = srv.URL

	ev := domain.Event{
		Provider: "mika",
		ID:       "BML_HCH3C0OH2F2",
		Name:     "BMW Berlin Marathon",
		Year:     2025,
	}

	res, err := c.Lookup(context.Background(), ev, "73664")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []struct {
		field string
		got   string
		want  string
	}{
		{"Runner", res.Runner, "Alexander Müller"},
		{"Bib", res.Bib, "73664"},
		{"NetTime", res.NetTime, "04:21:19"},
		{"GunTime", res.GunTime, "04:29:35"},
		{"Provider", res.Provider, "mika"},
	}
	for _, tc := range checks {
		if tc.got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.field, tc.got, tc.want)
		}
	}

	if res.OverallPlace != 24556 {
		t.Errorf("OverallPlace: got %d, want 24556", res.OverallPlace)
	}
	if res.GenderPlace != 17968 {
		t.Errorf("GenderPlace: got %d, want 17968", res.GenderPlace)
	}
	if !strings.Contains(res.SourceURL, "content=detail") {
		t.Errorf("SourceURL missing 'content=detail': %s", res.SourceURL)
	}
}

func TestLookup_Miss(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	c := New()
	c.BaseURL = srv.URL

	ev := domain.Event{
		Provider: "mika",
		ID:       "BML_HCH3C0OH2F2",
		Name:     "BMW Berlin Marathon",
		Year:     2025,
	}

	// The served detail.html is bib 73664; requesting 00000 triggers the bib guard.
	_, err := c.Lookup(context.Background(), ev, "00000")
	if !errors.Is(err, provider.ErrBibNotFound) {
		t.Errorf("expected ErrBibNotFound, got: %v", err)
	}
}
