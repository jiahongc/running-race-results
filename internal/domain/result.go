// internal/domain/result.go
package domain

// Split is one timing checkpoint within a race. All fields best-effort.
type Split struct {
	Label string // e.g. "5K", "HALF"
	Time  string // cumulative "HH:MM:SS"
	Pace  string // optional "MM:SS/km"
}

// Result is the unified, provider-independent race result.
type Result struct {
	Provider      string
	RaceName      string
	Year          int
	Runner        string
	Bib           string
	GunTime       string // "HH:MM:SS"
	NetTime       string // "HH:MM:SS"
	OverallPlace  int
	GenderPlace   int
	AgeGroup      string
	AgeGroupPlace int
	Splits        []Split
	SourceURL     string
}

// Event is a resolved race edition the resolver hands to an adapter.
type Event struct {
	Provider string // "nyrr","mika","athlinks","runsignup","raceroster","raceresult"
	Name     string
	Year     int
	ID       string // provider-specific event identifier
	BaseURL  string // provider-specific base, if needed
}
