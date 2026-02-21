# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build ./...          # compile all packages
go vet ./...            # static analysis
go run . <command>      # run without building binary
go build -o bin/whoop-garden .  # produce binary
```

No external dependencies — stdlib only.

## CLI Commands

```bash
go run . auth                        # OAuth2 flow → writes tokens.json
go run . daily [--date 2026-02-20]   # daily note → output/daily-YYYY-MM-DD.md
go run . weekly [--date 2026-02-20]  # weekly note → output/weekly-YYYY-WNN.md
go run . persona [--days 30]         # 30d persona section → stdout
go run . fetch-all [--days 30]       # batch write N daily notes
```

Output goes to `$OBSIDIAN_VAULT_PATH/Health/WHOOP/` if that env var is set, otherwise `./output/`.

## Architecture

The program is a thin pipeline: **fetch → model → render → write**.

```
main.go
  └─ loads .env, dispatches subcommand
       ├─ auth/auth.go      OAuth2 code flow (browser open + :3000 callback server)
       ├─ client/client.go  Authenticated HTTP GET with 429 retry
       ├─ fetch/fetch.go    Paginated API calls → DayData aggregate
       └─ render/render.go  text/template rendering + helpers
```

**Data flow for `daily`:**
1. `auth.RefreshIfNeeded()` loads/refreshes `tokens.json`
2. `client.NewClient(token)` wraps the token
3. `fetch.GetDayData(c, date)` concurrently fetches cycle, recovery, sleeps, workouts via goroutines, returns `DayData`
4. `render.RenderDaily(dayData, tmplPath)` executes `templates/daily.md.tmpl`
5. Output written to file

**Pagination pattern** (all list endpoints):
```go
for {
    params.Set("nextToken", nextToken)
    // decode PaginatedResponse[T]
    if page.NextToken == "" { break }
    nextToken = page.NextToken
}
```

**Templates** live in `templates/` and are loaded from disk at runtime. The template directory is resolved as: `$WHOOP_TEMPLATES_DIR` → `./templates/` (cwd) → `<binary_dir>/templates/`.

**FuncMap** helpers available in all templates: `millisToMinutes`, `recoveryColor`, `strainCategory`, `sportName`.

## Environment

Required in `.env` (auto-loaded on startup):
```
WHOOP_CLIENT_ID=...
WHOOP_CLIENT_SECRET=...
WHOOP_REDIRECT_URI=http://localhost:3000/callback
```

Optional:
```
OBSIDIAN_VAULT_PATH=/path/to/vault   # output destination
WHOOP_TEMPLATES_DIR=/path/to/tmpl    # override template location
```

## Key Files

| File | Purpose |
|------|---------|
| `internal/models/models.go` | All WHOOP v2 JSON structs + `SPORT_NAMES` map |
| `internal/auth/auth.go` | Token lifecycle: `StartAuthFlow`, `LoadTokens`, `SaveTokens`, `RefreshIfNeeded` |
| `internal/client/client.go` | `Client.Get(path, params)` — the only HTTP method needed |
| `internal/fetch/fetch.go` | `DayData` struct; `GetDayData` aggregates all data for one date |
| `internal/render/render.go` | `BuildWeekStats`, `RenderPersonaSection` (HRV linear regression) |
| `templates/*.md.tmpl` | Obsidian-flavored markdown templates |

## WHOOP API

Base URL: `https://api.prod.whoop.com/developer/v1`

Endpoints used: `/user/profile/basic`, `/user/measurement/body`, `/cycle`, `/recovery`, `/activity/sleep`, `/activity/workout`

Token endpoint: `https://api.prod.whoop.com/oauth/oauth2/token`
