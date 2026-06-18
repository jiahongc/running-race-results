// internal/provider/athlinks/athlinks.go
package athlinks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jiahongchen/race-results/internal/domain"
	"github.com/jiahongchen/race-results/internal/provider"
)

// Client is the Athlinks provider adapter.
type Client struct {
	BaseURL   string
	Token     string
	HTTP      *http.Client
	AlaskaURL string // alaska.athlinks.com host for the athlete endpoints
}

// New returns a Client configured for the Athlinks production API.
func New() *Client {
	return &Client{
		BaseURL:   "https://reignite-api.athlinks.com",
		Token:     os.Getenv("ATHLINKS_TOKEN"),
		HTTP:      &http.Client{Timeout: 15 * time.Second},
		AlaskaURL: "https://alaska.athlinks.com",
	}
}

// Name implements provider.Provider.
func (c *Client) Name() string { return "athlinks" }

// searchEntry is one element from the search results array.
type searchEntry struct {
	Bib           string `json:"bib"`
	DisplayName   string `json:"displayName"`
	EventCourseID int    `json:"eventCourseId"`
	Gender        string `json:"gender"`
	Age           int    `json:"age"`
}

// division represents one ranking division within an interval.
type division struct {
	Name          string `json:"name"`
	Rank          int    `json:"rank"`
	TotalAthletes int    `json:"totalAthletes"`
	Type          string `json:"type"`
}

// interval represents one timing point (finish or split).
type interval struct {
	Full             bool       `json:"full"`
	ChipTimeInMillis int64      `json:"chipTimeInMillis"`
	GunTimeInMillis  int64      `json:"gunTimeInMillis"`
	Divisions        []division `json:"divisions"`
}

// detailResponse is the per-athlete detail returned by the result endpoint.
type detailResponse struct {
	Bib         string     `json:"bib"`
	DisplayName string     `json:"displayName"`
	Intervals   []interval `json:"intervals"`
}

// formatSeconds renders an integer number of seconds as H:MM:SS (no leading
// zero on hours; minutes and seconds are zero-padded).
func formatSeconds(secs int64) string {
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

// searchEntries calls the /results/search endpoint with the given term and
// returns the raw slice of entries.
func (c *Client) searchEntries(ctx context.Context, ev domain.Event, term string) ([]searchEntry, error) {
	if c.Token == "" {
		return nil, errors.New("athlinks: ATHLINKS_TOKEN not set")
	}
	searchURL := fmt.Sprintf("%s/event/%s/results/search?from=0&limit=20&term=%s",
		c.BaseURL, ev.ID, url.QueryEscape(term))
	searchReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("athlinks: create search request: %w", err)
	}
	searchReq.Header.Set("Authorization", c.Token)
	searchReq.Header.Set("Origin", "https://www.athlinks.com")
	searchReq.Header.Set("Referer", "https://www.athlinks.com/")

	searchResp, err := c.HTTP.Do(searchReq)
	if err != nil {
		return nil, fmt.Errorf("athlinks: search request: %w", err)
	}
	defer searchResp.Body.Close()

	if searchResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("athlinks: search status %d", searchResp.StatusCode)
	}

	var entries []searchEntry
	if err := json.NewDecoder(searchResp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("athlinks: decode search response: %w", err)
	}
	return entries, nil
}

// SearchByName implements provider.NameSearcher. It uses the /results/search
// endpoint with the name as term and returns light Results (runner name + bib).
func (c *Client) SearchByName(ctx context.Context, ev domain.Event, name string) ([]domain.Result, error) {
	entries, err := c.searchEntries(ctx, ev, name)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(name)
	var out []domain.Result
	for _, e := range entries {
		if !strings.Contains(strings.ToLower(e.DisplayName), q) {
			continue
		}
		out = append(out, domain.Result{
			Provider: "athlinks",
			RaceName: ev.Name,
			Year:     ev.Year,
			Runner:   e.DisplayName,
			Bib:      e.Bib,
			SourceURL: fmt.Sprintf("%s/event/%s/results/search?term=%s",
				c.BaseURL, ev.ID, url.QueryEscape(name)),
		})
	}
	return out, nil
}

// Lookup implements provider.Provider.
func (c *Client) Lookup(ctx context.Context, ev domain.Event, bib string) (domain.Result, error) {
	// Step 1: search to resolve raceId (eventCourseId).
	entries, err := c.searchEntries(ctx, ev, bib)
	if err != nil {
		return domain.Result{}, err
	}

	var raceID int
	found := false
	for _, e := range entries {
		if e.Bib == bib {
			raceID = e.EventCourseID
			found = true
			break
		}
	}
	if !found {
		return domain.Result{}, provider.ErrBibNotFound
	}

	// Step 2: fetch per-athlete detail.
	detailURL := fmt.Sprintf("%s/event/%s/race/%d/bib/%s/result",
		c.BaseURL, ev.ID, raceID, bib)
	detailReq, err := http.NewRequestWithContext(ctx, http.MethodGet, detailURL, nil)
	if err != nil {
		return domain.Result{}, fmt.Errorf("athlinks: create detail request: %w", err)
	}
	detailReq.Header.Set("Authorization", c.Token)
	detailReq.Header.Set("Origin", "https://www.athlinks.com")
	detailReq.Header.Set("Referer", "https://www.athlinks.com/")

	detailResp, err := c.HTTP.Do(detailReq)
	if err != nil {
		return domain.Result{}, fmt.Errorf("athlinks: detail request: %w", err)
	}
	defer detailResp.Body.Close()

	if detailResp.StatusCode != http.StatusOK {
		return domain.Result{}, fmt.Errorf("athlinks: detail status %d", detailResp.StatusCode)
	}

	var dr detailResponse
	if err := json.NewDecoder(detailResp.Body).Decode(&dr); err != nil {
		return domain.Result{}, fmt.Errorf("athlinks: decode detail response: %w", err)
	}

	// Step 3: map detail → domain.Result.
	result := domain.Result{
		Provider:  "athlinks",
		RaceName:  ev.Name,
		Year:      ev.Year,
		Runner:    dr.DisplayName,
		Bib:       bib,
		SourceURL: detailURL,
	}

	for _, iv := range dr.Intervals {
		if !iv.Full {
			continue
		}
		result.NetTime = formatSeconds(iv.ChipTimeInMillis / 1000)
		result.GunTime = formatSeconds(iv.GunTimeInMillis / 1000)
		for _, div := range iv.Divisions {
			switch div.Type {
			case "overall":
				result.OverallPlace = div.Rank
			case "gender":
				result.GenderPlace = div.Rank
			case "other":
				if strings.ContainsAny(div.Name, "0123456789") {
					result.AgeGroup = div.Name
					result.AgeGroupPlace = div.Rank
				}
			}
		}
		break
	}

	return result, nil
}
