# Testing

## Running Tests

```bash
# Run all tests
go test ./...

# Skip slow tests (rate-limit retry involves real sleeps)
go test ./... -short

# Verbose output showing each test name
go test ./... -v

# Single package
go test ./internal/render/...
go test ./internal/client/...
go test ./internal/fetch/...

# Run a specific test
go test ./internal/render/... -run TestBuildWeekStats
```

## Coverage by Package

### `internal/render`

Pure functions — no mocks or I/O required.

| Test | What it covers |
|------|----------------|
| `TestMillisToMinutes` | Duration formatting including edge cases |
| `TestRecoveryColor` | All three zone boundaries (0, 34, 67, 100) |
| `TestStrainCategory` | All five category boundaries |
| `TestSportName` | Known ID, unknown ID fallback |
| `TestPrevNextDay` | Date navigation |
| `TestISOWeekStr` | ISO week string, prev/next week |
| `TestYearHelpers` | Cross-year ISO week boundary (Dec 31 → next year's week 1) |
| `TestPrimarySleep` | Longest non-nap, all-naps returns nil, empty returns nil |
| `TestNonNapSleeps` | Nap filtering, ordinal index assignment |
| `TestHRVTrendLabel` | Insufficient data, stable, improving, declining |
| `TestBuildWeekStats_*` | Empty input, full aggregation, PENDING_SCORE skipped, naps excluded |
| `TestRenderDaily` | Template execution smoke test with minimal template |
| `TestRenderPersonaSection_*` | Error on nil input, markdown output smoke test |

### `internal/client`

HTTP behavior via `net/http/httptest`.

| Test | What it covers |
|------|----------------|
| `TestGet_Success` | Bearer auth header forwarded, body returned |
| `TestGet_NotFound` | HTTP 404 → `ErrNotFound` sentinel |
| `TestGet_ServerError` | HTTP 500 → error returned |
| `TestGet_QueryParams` | Query params forwarded to server |
| `TestGet_PathAppended` | URL path correctly appended to base URL |
| `TestGet_RateLimitRetry` | 429 → retries → eventually succeeds (skipped under `-short`) |
| `TestGet_RateLimitExhausted` | Always skipped (requires injectable sleep to test fast) |

### `internal/fetch`

`ParseWhoopTime` and paginated collection functions via `httptest`.

| Test | What it covers |
|------|----------------|
| `TestParseWhoopTime` | WHOOP native format, RFC3339, invalid inputs |
| `TestGetCycles_Paginated` | Two-page response, records assembled in order, correct call count |
| `TestGetCycles_NotFound` | 404 → empty slice, no error |
| `TestGetCycles_EmptyPage` | Empty records page → empty slice |
| `TestGetCycles_QueryParams` | start/end forwarded to API |
| `TestGetSleeps_NotFound` | 404 → empty slice |
| `TestGetWorkouts_NotFound` | 404 → empty slice |
| `TestGetRecoveries_NotFound` | 404 → empty slice |

## Known Gaps

**`internal/auth`** — The auth package requires a live OAuth flow, a real
browser, and token files. It has no unit tests. Test the auth flow manually
after credential changes.

**`fetch.GetDayData` integration** — The concurrent fetch + cycle-matching
logic is not directly unit tested. It is covered indirectly by the individual
`GetCycles`, `GetRecoveries`, etc. tests. A full integration test would
require a multi-endpoint mock server.

**Rate-limit exhaustion** — `TestGet_RateLimitExhausted` is permanently
skipped because exhausting all 4 retries sleeps 1+2+4 = 7 seconds. Fixing
this requires making the sleep function injectable on `Client` (replace
`time.Sleep(backoff)` with a `sleepFn func(time.Duration)` field defaulting
to `time.Sleep`).

**Template rendering with real templates** — `TestRenderDaily` uses a minimal
stub template. The actual `templates/daily.md.tmpl` and `weekly.md.tmpl` are
not exercised by tests. A snapshot test against the real templates with
synthetic `DayData` would catch template regressions.

## Writing New Tests

Tests live in the same directory as the package they test, using the same
package name (giving access to unexported functions):

```
internal/render/render_test.go    package render
internal/client/client_test.go    package client
internal/fetch/fetch_test.go      package fetch
```

**For client tests** — use the internal struct directly since the test file is
in `package client`:

```go
c := &Client{
    accessToken: "tok",
    baseURL:     srv.URL,
    httpClient:  &http.Client{Timeout: 5 * time.Second},
}
```

**For fetch tests** — use `client.NewClientWithBaseURL(token, srv.URL)` to
point the client at a local `httptest` server:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(models.PaginatedResponse[models.Cycle]{
        Records:   []models.Cycle{{ID: 1}},
        NextToken: "",
    })
}))
defer srv.Close()
c := client.NewClientWithBaseURL("tok", srv.URL)
```

**For slow tests** — guard with `testing.Short()`:

```go
if testing.Short() {
    t.Skip("skipping: involves real sleep")
}
```
