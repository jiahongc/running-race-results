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
	"strings"
	"time"

	"github.com/jiahongchen/race-results/internal/domain"
	"github.com/jiahongchen/race-results/internal/provider"
)

// userAgent is a browser-like UA; the RaceResult hosts may reject the default
// Go-http-client UA.
const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// Client is the RaceResult provider adapter.
type Client struct {
	BaseURL     string // for the config call; default "https://my.raceresult.com"
	DataBaseURL string // for list calls; if empty, uses "https://{config.server}"
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

// cell holds one result-table cell. The list endpoint returns most cells as
// JSON strings but some as JSON numbers, so a permissive decoder is required —
// a strict [][]string would fail to decode the whole list on a single number.
type cell string

func (c *cell) UnmarshalJSON(b []byte) error {
	switch {
	case len(b) == 0 || string(b) == "null":
		*c = ""
	case b[0] == '"':
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*c = cell(s)
	default:
		*c = cell(strings.TrimSpace(string(b)))
	}
	return nil
}

// listResponse is the JSON envelope returned by the list endpoint.
type listResponse struct {
	DataFields []string            `json:"DataFields"`
	Data       map[string][][]cell `json:"data"`
}

var nonDigit = regexp.MustCompile(`\D`)

// get issues a GET with a browser User-Agent.
func (c *Client) get(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return c.HTTP.Do(req)
}

// Lookup implements provider.Provider. A RaceResult event exposes several
// result lists (teams/individual × gender), and a given bib lives in exactly
// one of them, so each list is queried until the bib is found.
func (c *Client) Lookup(ctx context.Context, ev domain.Event, bib string) (domain.Result, error) {
	configURL := fmt.Sprintf("%s/%s/results/config?lang=en", c.BaseURL, ev.ID)
	resp, err := c.get(ctx, configURL)
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
	if len(cfg.Tab.Config.Lists) == 0 {
		return domain.Result{}, fmt.Errorf("raceresult: no lists in config")
	}

	contestID := "1"
	for k := range cfg.Contests {
		contestID = k
		break
	}
	dataBase := c.DataBaseURL
	if dataBase == "" {
		dataBase = "https://" + cfg.Server
	}

	for _, l := range cfg.Tab.Config.Lists {
		if res, found := c.searchList(ctx, dataBase, ev, cfg.Key, l.Name, contestID, bib); found {
			return res, nil
		}
	}
	return domain.Result{}, provider.ErrBibNotFound
}

// searchList queries one result list and returns the mapped result if the bib
// is present. Per-list failures (network, non-200, decode, missing BIB column)
// return found=false so the caller moves on to the next list.
func (c *Client) searchList(ctx context.Context, dataBase string, ev domain.Event, key, listName, contestID, bib string) (domain.Result, bool) {
	listURL := fmt.Sprintf(
		"%s/%s/results/list?key=%s&listname=%s&page=results&contest=%s&r=leaders&l=50&fav=&openedGroups=%%7B%%7D&term=",
		dataBase, ev.ID, key, url.QueryEscape(listName), contestID,
	)
	resp, err := c.get(ctx, listURL)
	if err != nil {
		return domain.Result{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return domain.Result{}, false
	}
	var lr listResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return domain.Result{}, false
	}

	idx := make(map[string]int, len(lr.DataFields))
	for i, f := range lr.DataFields {
		idx[f] = i
	}
	bibIdx, ok := idx["BIB"]
	if !ok {
		return domain.Result{}, false
	}
	nameIdx, hasName := idx["AnzeigeName"]
	timeIdx, hasTime := idx["TIME1"]
	rankIdx, hasRank := idx["RANK2p"]
	// Only individual result lists carry the runner name + finish time. Team
	// and organisation lists also contain the bib but lack these columns, so
	// skip them — otherwise we'd return a partial (name/time-less) result.
	if !hasName || !hasTime {
		return domain.Result{}, false
	}

	for _, rows := range lr.Data {
		for _, row := range rows {
			if len(row) <= bibIdx || string(row[bibIdx]) != bib {
				continue
			}
			res := domain.Result{
				Provider:  "raceresult",
				RaceName:  ev.Name,
				Year:      ev.Year,
				Bib:       bib,
				SourceURL: "https://my.raceresult.com/" + ev.ID + "/results",
			}
			if hasName && len(row) > nameIdx {
				res.Runner = string(row[nameIdx])
			}
			if hasTime && len(row) > timeIdx {
				res.NetTime = string(row[timeIdx])
			}
			if hasRank && len(row) > rankIdx {
				if n, err := strconv.Atoi(nonDigit.ReplaceAllString(string(row[rankIdx]), "")); err == nil {
					res.OverallPlace = n
				}
			}
			return res, true
		}
	}
	return domain.Result{}, false
}
