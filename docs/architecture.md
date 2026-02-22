# Architecture

whoop-garden is a thin pipeline: **fetch → model → render → write**. Each
stage is a separate package with no circular dependencies.

## Package Map

```
main.go                       CLI entry, .env loading, subcommand dispatch
internal/
  auth/auth.go                OAuth2 flow, token save/load/refresh
  client/client.go            Authenticated HTTP GET, 429 retry/backoff
  fetch/fetch.go              Paginated API calls, DayData aggregation
  models/models.go            WHOOP v2 JSON structs, SPORT_NAMES map
  render/render.go            text/template rendering, FuncMap helpers
templates/
  daily.md.tmpl               Daily note template
  weekly.md.tmpl              Weekly summary template
```

## Data Flow

### `daily` command

```
main.runDaily()
  │
  ├─ auth.RefreshIfNeeded()
  │    └─ loads tokens.json, refreshes access token if expiring within 5 min
  │
  ├─ client.NewClient(token)
  │    └─ wraps token + 30s HTTP timeout
  │
  ├─ fetch.GetDayData(c, date)
  │    ├─ GetCycles(day 00:00 UTC, day+1 00:00 UTC)
  │    │    └─ pick cycles[0] as the day's anchor cycle
  │    │
  │    └─ concurrently (3 goroutines):
  │         ├─ GetRecoveries(cycleStart, cycleEnd)
  │         ├─ GetSleeps(cycleStart−24h, cycleEnd)   ← captures preceding night
  │         └─ GetWorkouts(cycleStart, cycleEnd)
  │              → DayData{Date, Cycle, Recovery, Sleeps[], Workouts[]}
  │
  ├─ render.RenderDaily(dayData, "templates/daily.md.tmpl")
  │    └─ text/template execution with FuncMap helpers
  │
  └─ os.WriteFile("<output>/<year>/daily-YYYY-MM-DD.md")
```

### `weekly` command

Same client/auth setup, then loops Mon–Sun calling `GetDayData` for each day,
passes the slice to `render.BuildWeekStats()` (aggregation) then
`render.RenderWeeklyFromStats()`.

### `persona` command

Loops over the last N days calling `GetDayData`, passes the full slice to
`render.RenderPersonaSection()` which aggregates inline and executes a
compiled-in template string. Output goes to the vault context-pack path if
`OBSIDIAN_VAULT_PATH` is set, otherwise stdout.

## WHOOP Cycle Alignment

WHOOP cycles do not align with calendar-day boundaries. A cycle starts when
the user wakes from their overnight sleep, which may be 6 AM or 10 AM
depending on the day. This means:

- Querying for "2026-02-20" means: find cycles whose `start` falls in
  `[2026-02-20T00:00:00Z, 2026-02-21T00:00:00Z)`
- The cycle's own `start`/`end` timestamps become the bounds for fetching
  associated recovery and workouts
- Sleep is fetched from `cycleStart − 24h` through `cycleEnd` to capture the
  overnight sleep that preceded the cycle

If no cycle is found for a calendar day, `GetDayData` returns an empty
`DayData{Date: day}` with nil Cycle/Recovery and empty slices. Templates
handle this gracefully with conditional rendering.

## Concurrency

`GetDayData` spawns three goroutines after the initial cycle fetch:

```go
recCh  := make(chan recoveriesResult, 1)
sleepCh := make(chan sleepResult, 1)
workCh  := make(chan workoutResult, 1)

go func() { recCh <- ... }()
go func() { sleepCh <- ... }()
go func() { workCh <- ... }()

rr := <-recCh
sr := <-sleepCh
wr := <-workCh
```

All channels are buffered (size 1) so goroutines never block even if the main
thread returns early on error. Errors from any goroutine are returned
immediately; the other two goroutines complete and their results are discarded.

## Pagination

Every WHOOP list endpoint uses the same token-based pagination pattern:

```go
for {
    params.Set("nextToken", nextToken)
    body, _ := c.Get(path, params)
    // decode PaginatedResponse[T]
    all = append(all, page.Records...)
    if page.NextToken == "" { break }
    nextToken = page.NextToken
}
```

A 404 response from any collection endpoint is treated as an empty result set
(not an error), since WHOOP returns 404 when there are no records in the
requested time range.

## Template Resolution

At startup, `templatesDir()` resolves in order:

1. `$WHOOP_TEMPLATES_DIR` env var
2. `./templates/` relative to cwd (used during `go run .` development)
3. `<binary_dir>/templates/` next to the compiled binary

The persona template is the exception — it is a compiled-in string constant
in `render/render.go` and does not depend on disk.

## Rate Limiting

- `client.Get` retries up to 3 times on HTTP 429 with exponential backoff:
  1 s → 2 s → 4 s
- `runFetchAll` and `runCatchUp` sleep 500 ms between each day's API calls

## Key Design Decisions

**Zero external dependencies** — pure stdlib. No module cache issues, no
supply chain risk, no version drift. The tradeoff is manual OAuth and HTTP
handling instead of using an SDK.

**Compiled-in persona template** — the persona output is a small, frequently
changed section. Keeping it as a string constant in `render.go` makes it easy
to edit without worrying about template file distribution.

**Year subdirectories** — output is organized as `<base>/<year>/filename.md`
to keep large Obsidian vaults navigable and match Obsidian's date-based folder
conventions.

**Recovery matched by cycle_id** — rather than assuming one recovery per day,
`GetDayData` matches recoveries to the specific cycle by `cycle_id`. This
handles edge cases where multiple recoveries appear in a query window.
