// internal/provider/raceresult/raceresult.go
package raceresult

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/jiahongchen/race-results/internal/domain"
	"github.com/jiahongchen/race-results/internal/provider"
)

// Client is the RaceResult provider adapter.
type Client struct {
	BaseURL     string // for config call; default "https://my.raceresult.com"
	DataBaseURL string // for list call; if empty, uses "https://{config.server}"
	HTTP        *http.Client
}

// New returns a Client configured for the RaceResult production API.
func New() *Client {
	return &Client{
		BaseURL: "https://my.raceresult.com",
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

// Name implements provider.Provider.
func (c *Client) Name() string { return "raceresult" }

// configResponse is the JSON envelope returned by the config endpoint.
type configResponse struct {
	Key      string            `json:"key"`
	Server   string            `json:"server"`
	Contests map[string]string `json:"contests"`
	Tab      struct {
		Config struct {
			Lists []struct {
				Name string `json:"Name"`
			} `json:"Lists"`
		} `json:"Config"`
	} `json:"Tab"`
}

// listResponse is the JSON envelope returned by the list endpoint.
type listResponse struct {
	DataFields []string                `json:"DataFields"`
	Data       map[string][][]string   `json:"data"`
}

var nonDigit = regexp.MustCompile(`\D`)

// Lookup implements provider.Provider.
func (c *Client) Lookup(ctx context.Context, ev domain.Event, bib string) (domain.Result, error) {
	// Step 1: fetch config.
	configURL := fmt.Sprintf("%s/%s/results/config?lang=en", c.BaseURL, ev.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, configURL, nil)
	if err != nil {
		return domain.Result{}, fmt.Errorf("raceresult: create config request: %w", err)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return domain.Result{}, fmt.Errorf("raceresult: config request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.Result{}, fmt.Errorf("raceresult: config status %d", resp.StatusCode)
	}

	var cfg configResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return domain.Result{}, fmt.Errorf("raceresult: decode config: %w", err)
	}

	// Extract first list name.
	if len(cfg.Tab.Config.Lists) == 0 {
		return domain.Result{}, fmt.Errorf("raceresult: no lists in config")
	}
	listName := cfg.Tab.Config.Lists[0].Name

	// Extract first contest ID.
	contestID := "1"
	for k := range cfg.Contests {
		contestID = k
		break
	}

	// Step 2: fetch results list.
	dataBase := c.DataBaseURL
	if dataBase == "" {
		dataBase = "https://" + cfg.Server
	}

	listURL := fmt.Sprintf(
		"%s/%s/results/list?key=%s&listname=%s&page=results&contest=%s&r=leaders&l=50&fav=&openedGroups=%%7B%%7D&term=",
		dataBase,
		ev.ID,
		cfg.Key,
		url.QueryEscape(listName),
		contestID,
	)

	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return domain.Result{}, fmt.Errorf("raceresult: create list request: %w", err)
	}

	resp2, err := c.HTTP.Do(req2)
	if err != nil {
		return domain.Result{}, fmt.Errorf("raceresult: list request: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		return domain.Result{}, fmt.Errorf("raceresult: list status %d", resp2.StatusCode)
	}

	var lr listResponse
	if err := json.NewDecoder(resp2.Body).Decode(&lr); err != nil {
		return domain.Result{}, fmt.Errorf("raceresult: decode list: %w", err)
	}

	// Build name→index map from DataFields.
	fieldIdx := make(map[string]int, len(lr.DataFields))
	for i, f := range lr.DataFields {
		fieldIdx[f] = i
	}

	bibIdx, hasBib := fieldIdx["BIB"]
	nameIdx, hasName := fieldIdx["AnzeigeName"]
	timeIdx, hasTime := fieldIdx["TIME1"]
	rankIdx, hasRank := fieldIdx["RANK2p"]

	if !hasBib || !hasName || !hasTime || !hasRank {
		return domain.Result{}, fmt.Errorf("raceresult: required DataFields missing (got %v)", lr.DataFields)
	}

	// Search all groups for a matching bib.
	for _, rows := range lr.Data {
		for _, row := range rows {
			if len(row) <= bibIdx || row[bibIdx] != bib {
				continue
			}

			name := ""
			if len(row) > nameIdx {
				name = row[nameIdx]
			}
			netTime := ""
			if len(row) > timeIdx {
				netTime = row[timeIdx]
			}
			rank := 0
			if len(row) > rankIdx {
				digits := nonDigit.ReplaceAllString(row[rankIdx], "")
				if n, err := strconv.Atoi(digits); err == nil {
					rank = n
				}
			}

			return domain.Result{
				Provider:     "raceresult",
				RaceName:     ev.Name,
				Year:         ev.Year,
				Runner:       name,
				Bib:          bib,
				NetTime:      netTime,
				OverallPlace: rank,
				SourceURL:    "https://my.raceresult.com/" + ev.ID + "/results",
			}, nil
		}
	}

	return domain.Result{}, provider.ErrBibNotFound
}
