# Running Race Results

A command-line tool that looks up a runner's race result by **race name + bib number**, across multiple timing providers. The race name is fuzzy-matched, so `"berlin marathon"` resolves to the right event.

```
race-results lookup "<race name>" <bib> [--year YYYY] [--date YYYY-MM-DD] [--json]
```

```console
$ race-results lookup "berlin marathon" 73664 --year 2025
Race           BMW Berlin Marathon 2025
Runner         Alexander Müller
Bib            73664
Net time       04:21:19
Gun time       04:29:35
Overall place  24556
Gender place   17968
Source         https://berlin.r.mikatiming.com/2025/?content=detail&...
```

## Providers

There is no single API for race results — each timing platform is different. This tool wraps several behind one interface.

| Provider | Coverage | Status |
|----------|----------|--------|
| **NYRR** | New York Road Runners events | ✅ live |
| **Mika Timing** | Berlin + World Marathon Majors (Boston, Chicago, London, Tokyo, …) | ✅ live |
| **Athlinks** | Aggregator (many events worldwide) | ✅ live — needs `ATHLINKS_TOKEN` |
| **RaceResult** | Events on `my.raceresult.com` | ✅ live |
| **RunSignup** | Large US race base | 🔜 planned (free API key) |
| **Race Roster** | `results.raceroster.com` events | 🔜 planned |

## How it works

```
race name + bib
      │
      ▼
  catalog ──▶ fuzzy resolver ──▶ (provider, event, year)
                                        │
                                        ▼
                                 provider adapter ──▶ live API / page
                                        │
                                        ▼
                              unified Result ──▶ table / JSON
```

- **Catalog** (`internal/catalog/catalog.json`): a bundled map of known races → provider + event id + name aliases. Extend it by adding an entry — no code change needed.
- **Resolver** (`internal/resolve`): normalizes the query (drops sponsor prefixes like `TCS`/`BMW`), then scores it against the catalog with token-overlap + edit distance. `--year` / `--date` disambiguate the edition.
- **Provider adapters** (`internal/provider/*`): one per timing platform, each implementing a common `Provider` interface (`Lookup(event, bib) → Result`).
- **Result** (`internal/domain`): a unified shape (runner, bib, net/gun time, places, age group, splits, source URL). Missing fields are omitted, never faked.

## Install

Requires **Go 1.26.4+**.

```bash
go build -o race-results ./cmd/race-results
./race-results --help
```

## Usage

```bash
# Look up by race name + bib
race-results lookup "mini 10k" 19 --year 2026

# Disambiguate the edition by date (year is derived)
race-results lookup "berlin marathon" 73664 --date 2025-09-28

# Machine-readable output
race-results lookup "berlin marathon" 73664 --year 2025 --json
```

If a race name matches multiple events, the tool lists the candidates; refine with `--year` or a fuller name.

## Configuration

Secrets are read from the environment (never hardcoded). Put them in a local `.env` (gitignored):

| Variable | Used by | Notes |
|----------|---------|-------|
| `ATHLINKS_TOKEN` | Athlinks | A `Bearer …` token from the Athlinks frontend. Short-lived (~2h). |
| `RUNSIGNUP_API_KEY` | RunSignup (planned) | Free key from <https://runsignup.com/API>. |

```bash
# .env
ATHLINKS_TOKEN="Bearer eyJ…"
```

## Development

```bash
go test ./...        # unit tests (offline — adapters run against recorded fixtures)
go vet ./...
gofmt -l .           # must be empty
```

- Adapters are tested against real recorded responses in `testdata/fixtures/<provider>/`, served by an in-process `httptest` server — no network in unit tests.
- Provider request/response shapes are documented in [`docs/providers/contracts.md`](docs/providers/contracts.md).
- **Adding a provider:** implement `provider.Provider` in `internal/provider/<name>`, register it in `cmd/race-results/main.go`, add catalog entries, and a fixture-backed test.

## Notes

- Lookups are **by bib**, not by runner name. Live tracking and analytics are out of scope.
- Test fixtures contain publicly published race results (real names/times) used purely as recorded API responses for offline tests.
