# Provider API Contracts

Captured request/response shapes per provider. Each adapter (Phase 2) is built
from the matching `testdata/fixtures/<provider>/` fixture + the note below.

> Status: Phase 0 discovery. Fill from real captures only — never fabricate.

## NYRR

### Request

```
POST https://rmsprodapi.nyrr.org/api/v2/runners/finishers-filter
Content-Type: application/json
```

Body (`testdata/fixtures/nyrr/request.json`):
```json
{
  "eventCode": "26MINI",
  "searchString": "Smith",
  "gender": null,
  "ageFrom": null,
  "ageTo": null,
  "sortColumn": "overallTime",
  "sortDescending": false,
  "pageIndex": 1,
  "pageSize": 20
}
```

- Event code comes from the URL: `results.nyrr.org/races/{eventCode}/results`
- No auth token needed — open API with standard browser headers
- Pagination: `pageIndex` (1-based), `pageSize`
- Bib lookup: set `searchString` to bib number (numeric string)

### Response → Result mapping

Response (`testdata/fixtures/nyrr/search.json`):
```json
{
  "totalItems": 40,
  "items": [
    {
      "runnerId": 52475309,
      "firstName": "Rachel",
      "lastName": "Smith",
      "bib": "19",
      "age": 34,
      "gender": "W",
      "city": "Flagstaff",
      "countryCode": "USA",
      "stateProvince": "AZ",
      "overallPlace": 20,
      "overallTime": "0:33:48",
      "pace": "05:27",
      "genderPlace": 20,
      "ageGradeTime": "33:26",
      "ageGradePlace": 23,
      "ageGradePercent": 90.76,
      "racesCount": 7
    }
  ]
}
```

Field mapping:
- `bib` → BIB
- `firstName` + `lastName` → name
- `overallPlace` → overall rank
- `genderPlace` → gender rank
- `overallTime` → finish time (gun, format `H:MM:SS`)
- `pace` → pace per mile (format `MM:SS`)
- `age`, `gender` ("W"/"M"), `city`, `countryCode`, `stateProvince` → athlete metadata
- `ageGradePercent` → age grade %

---

## Mika

Site: `https://berlin.r.mikatiming.com/` (Berlin Marathon 2025)

### Request

**Search (POST — returns full HTML page with embedded runner list):**
```
POST https://berlin.r.mikatiming.com/?event=BML_HCH3C0OH2F2&pid=search
Content-Type: application/x-www-form-urlencoded

search[name]=Müller&search[start_no]=&search[nation]=&Search=Search
```

- `event` param: event code in URL (e.g., `BML_HCH3C0OH2F2` for Berlin Marathon 2025)
- `pid=search`: triggers search results page
- Returns HTML (not JSON); runner list in `<ul class="list-group list-group-multicolumn">`

**Detail page (GET):**
```
GET https://berlin.r.mikatiming.com/?content=detail&fpid=search&pid=search&idp={runner_id}&lang=EN_CAP&event={event_code}
```

- `idp`: unique runner ID extracted from search result links
- Returns full HTML detail page with split table

**Autocomplete AJAX (GET — returns JSON, NOT used for structured results):**
```
GET https://berlin.r.mikatiming.com/index.php?content=ajax2&func=getSearchResult&event={event}&lang=EN&search[name]=Müller
```

### Response → Result mapping

Search HTML (`testdata/fixtures/mika/search.html`):
- Runner rows in `<ul class="list-group list-group-multicolumn">`
- Each runner link: `?content=detail&fpid=search&pid=search&idp={runner_id}&lang=EN_CAP&event={event_code}`
- Extract `idp` parameter for detail lookup

Detail HTML (`testdata/fixtures/mika/detail.html`), CSS class selectors:
- `td.f-__fullname` → full name
- `td.f-start_no_text` → bib
- `td.f-time_finish_netto` → chip time (net)
- `td.f-time_finish_brutto` → gun time
- `td.f-place_all` → overall place
- `td.f-place_nosex` → gender place
- `td.f-place_age` → age group place
- Split times at checkpoints: `td.f-time_15000`, `td.f-time_half`, `td.f-time_30000`, `td.f-time_40000`, etc.

Example (Alexander Müller, bib 73664, Berlin Marathon 2025):
- Net time: 04:21:19, Gun time: 04:29:35
- Overall: 24556, Gender: 17968, Age group: 3322

---

## Race Roster

Site: `https://results.raceroster.com/v3/events/{eventUniqueCode}`

Event discovery: event page at `raceroster.com/events/{year}/{eventId}/{slug}` contains a link to `results.raceroster.com/v3/events/{eventUniqueCode}`.

### Request

**Participant search:**
```
GET https://results.raceroster.com/v2/api/events/{eventUniqueCode}/participant-search?phrase={name}
```
Returns: `{ data: { exact: [...], other: [...] } }` — each item has `id`, `firstName`, `lastName`, `bib`, `gender`, `age`, `resultSubEventId`, `teamName`.

**Sub-event metadata:**
```
GET https://results.raceroster.com/v2/api/events/{eventUniqueCode}/sub-events/{subEventId}
```
Returns sub-event config: `{ id, name, distance, distanceUnit, resultCount, columns, filters }`.

**Individual result (flat):**
```
GET https://results.raceroster.com/v2/api/results/{participantId}
```
Returns flat result object.

**Individual result (detail with stats/certificates):**
```
GET https://results.raceroster.com/v2/api/events/{eventUniqueCode}/detail/{participantId}
```
Returns: `{ result: {...}, participant: {...}, finisherCertificate: {...}, stats: {...}, segmentResults: null }`.

Captured event: 2024 TCS Toronto Waterfront Marathon — Virtual Half Marathon
- `eventUniqueCode`: `4p3khwy5ujzf2v33`
- `subEventId`: `213657`
- `participantId`: `xf6gytfz69fm6mj6` (Nancy Stonos-Smith)

### Response → Result mapping

Fixtures: `testdata/fixtures/raceroster/search.json`, `testdata/fixtures/raceroster/result.json`

From `/v2/api/results/{participantId}`:
- `bib` → BIB (empty string for virtual events)
- `firstName` + `lastName` → name
- `overallPlace` → overall rank (of `genderPlaceCount` total in gender)
- `genderPlace` → gender rank
- `gunTime` → finish time (format `H:MM:SS`)
- `gunTimeSec` → finish time in seconds
- `overallPace` → pace per km (format `MM:SS`)
- `overallPaceUnits` → `"min/km"` or `"min/mi"`
- `age`, `gender` ("Female"/"Male"), `fromCity`, `fromProvState`, `fromCountry` → athlete metadata
- `teamName` → team/club
- `raceStatus`: `"COMPLETE"` | `"DNF"` | `"DNS"`

---

## RaceResult

Site: `https://my.raceresult.com/{eventId}/results`

Event discovery: `GET /RREvents/list?group=0&user=0&userID=0&geoLocation=IP&lang=en&modes=topResults` returns array of event objects with `id`, `name`, `dateFrom`, `location`, `countryCode`.

### Request

**Config (required first — returns key + list names):**
```
GET https://my.raceresult.com/{eventId}/results/config?lang=en
```
Returns: `{ key, contests, splits, eventname, EventOver, server, Tab: { Config: { Lists: [...] } } }`.
- `server`: hostname for subsequent data calls (e.g., `my-us-1.raceresult.com`)
- `Tab.Config.Lists[].Name`: list name to pass to the list endpoint

**Results list:**
```
GET https://{server}/{eventId}/results/list?key={key}&listname={encodedListName}&page=results&contest={contestId}&r=leaders&l=10&fav=&openedGroups=%7B%7D&term=
```
- `listname`: URL-encoded list name from config (e.g., `Ergebnislisten%7CInternet-einzel%20-%20Frauen`)
- `contest`: contest ID (from `config.contests` keys, typically `"1"`)
- `term`: name/bib search filter (empty for full list)
- `page`: `"results"` for initial load; numeric for pagination

**Participant detail:**
```
GET https://{server}/{eventId}/{detailsTabName}/view?lang=en&noVisitor=1&mid=0&standalone=false&pid={pid}
```
- `detailsTabName`: from `config.Tab.Config.StandardDetails` (e.g., `"details0"`)
- `pid`: participant ID (= BIB, same as first two DataFields columns in list response)

Captured event: 17. REWE Team Challenge Dresden, 2026-06-17
- `eventId`: `390537`
- `key`: `93941475da0e781fdf01c051062b7423`
- `server`: `my-us-1.raceresult.com`

Fixtures: `testdata/fixtures/raceresult/config.json`, `testdata/fixtures/raceresult/results.json`, `testdata/fixtures/raceresult/detail.json`

### Response → Result mapping

List response: `{ DataFields: [...], data: { "#group_name": [[row], [row], ...] }, mid }`.
- `DataFields` names columns; `data` rows are parallel arrays
- Typical columns: `BIB`, `ID` (= pid), `RANK2p` (place label e.g. "1."), `AnzeigeName` (display name), `CLUB`, `Organisation`, `TIME1` (finish time)
- `ID` column (index 1) = `pid` for detail lookup

Field mapping from list row:
- `row[0]` (BIB) → BIB
- `row[1]` (ID) → pid for detail lookup
- `row[2]` (RANK2p) → rank display ("1.", "2.", ...)
- `row[3]` (AnzeigeName) → full name
- `row[4]` (CLUB) → team name
- `row[6]` (TIME1) → finish time (format `H:MM:SS`)

Detail response: `{ Data: { SplitsAndLegs: { Splits, Legs }, Fields, Certificates, Photos }, PID, MID, Server }`.
- `Data.Fields`: null in this event (splits not exposed); when present, contains split times
- `Data.Photos`: array of photo URLs (Sportograf etc.)

---

## RunSignup
_(you: register free API key)_
### Request
### Response → Result mapping

## Athlinks
_(you: token capture)_
### Request
### Response → Result mapping
