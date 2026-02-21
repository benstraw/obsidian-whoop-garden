# Plan: Scaffold whoop-garden Go CLI

## Context
Building a personal Go CLI tool that pulls data from WHOOP v2 API and generates markdown files for an Obsidian digital garden. The project directory is currently empty (only `.env` exists). Full spec is provided by the user — no ambiguity on requirements.

## Files to Create (in order)

### 1. `go.mod`
- Module: `github.com/benstraw/whoop-garden`
- Go 1.21+
- No external dependencies (stdlib only — manual HTTP for OAuth)

### 2. `.gitignore`
- Ignore: `.env`, `tokens.json`, `/bin/`, `/output/`

### 3. `internal/models/models.go`
All WHOOP v2 structs with JSON tags:
- `UserProfile`, `BodyMeasurements`
- `Cycle` / `CycleScore`
- `Recovery` / `RecoveryScore`
- `Sleep` / `SleepScore` / `SleepStageSummary` / `SleepNeeded`
- `Workout` / `WorkoutScore` / `ZoneDuration`
- `PaginatedResponse[T any]` generic wrapper
- `SPORT_NAMES map[int]string` (all ~60 entries)

### 4. `internal/auth/auth.go`
- `TokenResponse` struct (with computed `ExpiresAt time.Time`)
- `StartAuthFlow()` — builds auth URL with scopes, opens browser via `exec.Command("open", url)`, starts `http.Server` on `:3000`, captures callback code, exchanges for tokens via POST, saves tokens
- `SaveTokens(tokens TokenResponse)` — writes `tokens.json` with `json.Marshal`
- `LoadTokens() (TokenResponse, error)` — reads `tokens.json`
- `RefreshIfNeeded() (string, error)` — checks `ExpiresAt`, refreshes if needed

### 5. `internal/client/client.go`
- `Client` struct with `accessToken` and `baseURL`
- `NewClient(token string) *Client`
- `Get(path string, params url.Values) ([]byte, error)` — adds Bearer header, handles 429 with 1s backoff + single retry

### 6. `internal/fetch/fetch.go`
- `DayData` struct aggregating cycle, recovery, sleeps, workouts for one day
- `GetUserProfile(c *client.Client) (*models.UserProfile, error)`
- `GetBodyMeasurements(c *client.Client) (*models.BodyMeasurements, error)`
- `GetCycles/Recoveries/Sleeps/Workouts(c, start, end)` — paginated (loop on `next_token`)
- `GetDayData(c, date)` — fetches all four, filters to relevant records for that day

### 7. `internal/render/render.go`
- `RenderDaily(data fetch.DayData, tmplPath string) (string, error)` — parses file template
- `RenderWeekly(data []fetch.DayData, tmplPath string) (string, error)` — aggregates stats
- `RenderPersonaSection(data []fetch.DayData) (string, error)` — uses inline const template, calculates 30d rolling averages + HRV trend slope
- Helpers (used in templates via `FuncMap`): `MillisToMinutes`, `RecoveryColor`, `StrainCategory`, `SportName`

### 8. `templates/daily.md.tmpl`
- YAML frontmatter (date, tags)
- Summary placeholder
- Recovery, Sleep, Strain, Workouts sections

### 9. `templates/weekly.md.tmpl`
- Week range header
- Aggregate stats table
- Green/Yellow/Red day tally
- Chronological workout list
- Best/worst recovery day callouts

### 10. `main.go`
- Manual `.env` parser (read file, split on `=`, call `os.Setenv`)
- Subcommand dispatch: `auth`, `daily [--date]`, `weekly`, `persona`, `fetch-all [--days]`
- Output path logic: `$OBSIDIAN_VAULT_PATH/Health/WHOOP/` or `./output/`
- `os.MkdirAll` on output dir before writing

## Key Implementation Notes

**Pagination**: WHOOP v2 returns `next_token` string in paginated responses. Loop: set `nextToken` query param, break when response `next_token` is empty string.

**Auth callback server**: Use a channel to pass the auth code from the HTTP handler goroutine back to `StartAuthFlow`. Shut down server via `server.Shutdown(ctx)` after code received.

**Token refresh**: POST to `https://api.prod.whoop.com/oauth/oauth2/token` with `grant_type=refresh_token`, `client_id`, `client_secret`, `refresh_token`. Same endpoint as initial exchange.

**ExpiresAt calculation**: `time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)` at token save time.

**HRV trend** in persona: linear regression slope over 30d of HRV values — positive = improving, negative = declining, near-zero = stable.
