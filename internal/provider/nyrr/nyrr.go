// internal/provider/nyrr/nyrr.go
package nyrr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jiahongchen/race-results/internal/domain"
	"github.com/jiahongchen/race-results/internal/provider"
)

// Client is the NYRR provider adapter.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a Client configured for the NYRR production API.
func New() *Client {
	return &Client{
		BaseURL: "https://rmsprodapi.nyrr.org",
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

// Name implements provider.Provider.
func (c *Client) Name() string { return "nyrr" }

// searchRequest is the body sent to the finishers-filter endpoint.
type searchRequest struct {
	EventCode     string `json:"eventCode"`
	SearchString  string `json:"searchString"`
	PageIndex     int    `json:"pageIndex"`
	PageSize      int    `json:"pageSize"`
	SortColumn    string `json:"sortColumn"`
	SortDescending bool  `json:"sortDescending"`
}

// searchResponse is the JSON envelope returned by the API.
type searchResponse struct {
	TotalItems int    `json:"totalItems"`
	Items      []item `json:"items"`
}

type item struct {
	RunnerID      int     `json:"runnerId"`
	FirstName     string  `json:"firstName"`
	LastName      string  `json:"lastName"`
	Bib           string  `json:"bib"`
	Age           int     `json:"age"`
	Gender        string  `json:"gender"`
	City          string  `json:"city"`
	StateProvince string  `json:"stateProvince"`
	CountryCode   string  `json:"countryCode"`
	OverallPlace  int     `json:"overallPlace"`
	OverallTime   string  `json:"overallTime"`
	Pace          string  `json:"pace"`
	GenderPlace   int     `json:"genderPlace"`
	AgeGradeTime  string  `json:"ageGradeTime"`
	AgeGradePlace int     `json:"ageGradePlace"`
	AgeGradePercent float64 `json:"ageGradePercent"`
	RacesCount    int     `json:"racesCount"`
}

// Lookup implements provider.Provider.
func (c *Client) Lookup(ctx context.Context, ev domain.Event, bib string) (domain.Result, error) {
	reqBody := searchRequest{
		EventCode:      ev.ID,
		SearchString:   bib,
		PageIndex:      1,
		PageSize:       50,
		SortColumn:     "overallTime",
		SortDescending: false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return domain.Result{}, fmt.Errorf("nyrr: marshal request: %w", err)
	}

	url := c.BaseURL + "/api/v2/runners/finishers-filter"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return domain.Result{}, fmt.Errorf("nyrr: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return domain.Result{}, fmt.Errorf("nyrr: http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.Result{}, fmt.Errorf("nyrr: unexpected status %d", resp.StatusCode)
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return domain.Result{}, fmt.Errorf("nyrr: decode response: %w", err)
	}

	for _, it := range sr.Items {
		if it.Bib != bib {
			continue
		}
		return domain.Result{
			Provider:     "nyrr",
			RaceName:     ev.Name,
			Year:         ev.Year,
			Runner:       it.FirstName + " " + it.LastName,
			Bib:          it.Bib,
			NetTime:      it.OverallTime,
			OverallPlace: it.OverallPlace,
			GenderPlace:  it.GenderPlace,
			SourceURL:    "https://results.nyrr.org/races/" + ev.ID + "/results",
		}, nil
	}

	return domain.Result{}, provider.ErrBibNotFound
}
