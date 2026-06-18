// internal/provider/mika/mika.go
package mika

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html"

	"github.com/jiahongchen/race-results/internal/domain"
	"github.com/jiahongchen/race-results/internal/provider"
)

// Client is the Mika Timing provider adapter.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a Client configured for the Mika Timing production host.
func New() *Client {
	return &Client{
		BaseURL: "https://berlin.r.mikatiming.com",
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

// Name implements provider.Provider.
func (c *Client) Name() string { return "mika" }

// reIDP matches the idp query param inside a content=detail href.
var reIDP = regexp.MustCompile(`content=detail[^"']*idp=([^&"']+)`)

// Lookup implements provider.Provider.
func (c *Client) Lookup(ctx context.Context, ev domain.Event, bib string) (domain.Result, error) {
	base := c.BaseURL
	if ev.BaseURL != "" {
		base = ev.BaseURL
	}

	// Step 1: POST search by bib.
	searchURL := fmt.Sprintf("%s/?event=%s&pid=search", base, url.QueryEscape(ev.ID))
	body := url.Values{
		"search[name]":     {""},
		"search[start_no]": {bib},
		"search[nation]":   {""},
		"Search":           {"Search"},
	}
	searchReq, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL, strings.NewReader(body.Encode()))
	if err != nil {
		return domain.Result{}, fmt.Errorf("mika: create search request: %w", err)
	}
	searchReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	searchResp, err := c.HTTP.Do(searchReq)
	if err != nil {
		return domain.Result{}, fmt.Errorf("mika: search request: %w", err)
	}
	defer searchResp.Body.Close()

	searchHTML, err := io.ReadAll(searchResp.Body)
	if err != nil {
		return domain.Result{}, fmt.Errorf("mika: read search response: %w", err)
	}

	// Step 2: Extract first idp from a content=detail link.
	m := reIDP.FindSubmatch(searchHTML)
	if m == nil {
		return domain.Result{}, provider.ErrBibNotFound
	}
	idp := string(m[1])

	// Step 3: GET detail page.
	detailURL := fmt.Sprintf("%s/?content=detail&fpid=search&pid=search&idp=%s&lang=EN_CAP&event=%s",
		base, url.QueryEscape(idp), url.QueryEscape(ev.ID))
	detailReq, err := http.NewRequestWithContext(ctx, http.MethodGet, detailURL, nil)
	if err != nil {
		return domain.Result{}, fmt.Errorf("mika: create detail request: %w", err)
	}

	detailResp, err := c.HTTP.Do(detailReq)
	if err != nil {
		return domain.Result{}, fmt.Errorf("mika: detail request: %w", err)
	}
	defer detailResp.Body.Close()

	doc, err := html.Parse(detailResp.Body)
	if err != nil {
		return domain.Result{}, fmt.Errorf("mika: parse detail HTML: %w", err)
	}

	// Step 4: Parse detail cells.
	fullName := tdText(doc, "f-__fullname")
	parsedBib := tdText(doc, "f-start_no_text")
	netTime := tdText(doc, "f-time_finish_netto")
	gunTime := tdText(doc, "f-time_finish_brutto")
	// f-place_all  = Place (M/W/D) = gender place
	// f-place_nosex = Place (Total) = overall place
	genderPlaceStr := tdText(doc, "f-place_all")
	overallPlaceStr := tdText(doc, "f-place_nosex")

	// Step 5: Bib guard — prevent returning a wrong runner.
	if parsedBib != bib {
		return domain.Result{}, provider.ErrBibNotFound
	}

	overallPlace := parsePlace(overallPlaceStr)
	genderPlace := parsePlace(genderPlaceStr)

	return domain.Result{
		Provider:     "mika",
		RaceName:     ev.Name,
		Year:         ev.Year,
		Runner:       parseName(fullName),
		Bib:          parsedBib,
		NetTime:      netTime,
		GunTime:      gunTime,
		OverallPlace: overallPlace,
		GenderPlace:  genderPlace,
		SourceURL:    detailURL,
	}, nil
}

// tdText walks the HTML tree and returns the trimmed text of the first <td>
// whose class attribute contains the given token.
func tdText(n *html.Node, classToken string) string {
	if n.Type == html.ElementNode && n.Data == "td" {
		for _, a := range n.Attr {
			if a.Key == "class" && containsToken(a.Val, classToken) {
				return strings.TrimSpace(textContent(n))
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if v := tdText(c, classToken); v != "" {
			return v
		}
	}
	return ""
}

// textContent returns the concatenated text of all text nodes under n.
func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}

// containsToken reports whether s contains tok as a whitespace-separated token.
func containsToken(s, tok string) bool {
	for _, t := range strings.Fields(s) {
		if t == tok {
			return true
		}
	}
	return false
}

// parseName converts "Last, First (Country)" to "First Last".
// Falls back to the raw string if the format is not recognised.
func parseName(raw string) string {
	// Strip parenthetical country suffix, e.g. " (GER)".
	if idx := strings.LastIndex(raw, "("); idx >= 0 {
		raw = strings.TrimSpace(raw[:idx])
	}
	// Split on first comma.
	parts := strings.SplitN(raw, ",", 2)
	if len(parts) == 2 {
		last := strings.TrimSpace(parts[0])
		first := strings.TrimSpace(parts[1])
		return first + " " + last
	}
	return raw
}

// parsePlace strips non-digit characters (commas, spaces) and converts to int.
func parsePlace(s string) int {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return 0
	}
	v, _ := strconv.Atoi(b.String())
	return v
}
