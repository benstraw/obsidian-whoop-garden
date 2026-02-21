# whoop-garden

Pulls data from the [WHOOP v2 API](https://developer.whoop.com) and renders
structured Obsidian markdown notes — daily summaries, weekly rollups, and a
rolling 30-day health persona section.

No external dependencies. stdlib only.

## Setup

### 1. Register a WHOOP app

Go to [developer.whoop.com](https://developer.whoop.com), create an app, and
set the redirect URI to `http://localhost:3000/callback`.

### 2. Configure environment

```bash
cp .env.example .env   # then fill in your values
```

`.env`:
```
WHOOP_CLIENT_ID=your_client_id
WHOOP_CLIENT_SECRET=your_client_secret
WHOOP_REDIRECT_URI=http://localhost:3000/callback

# Optional — route output directly to your vault
OBSIDIAN_VAULT_PATH=/path/to/your/obsidian/vault
```

If `OBSIDIAN_VAULT_PATH` is set, files are written to `Health/WHOOP/` inside
your vault. Otherwise they go to `./output/`.

### 3. Authenticate

```bash
go run . auth
```

Opens a browser to WHOOP's OAuth page. After approving, tokens are saved to
`tokens.json` automatically. Tokens auto-refresh — you should only need to do
this once.

## Usage

```bash
# Daily note for today
go run . daily

# Daily note for a specific date
go run . daily --date 2026-02-19

# Weekly summary (Mon–Sun of the current week)
go run . weekly

# Weekly summary for the week containing a given date
go run . weekly --date 2026-02-17

# 30-day persona section → stdout (paste into your health persona note)
go run . persona

# Custom window
go run . persona --days 60

# Backfill N days of daily notes
go run . fetch-all --days 90
```

## Build

```bash
go build ./...                          # compile
go build -o bin/whoop-garden .          # produce binary
go vet ./...                            # static analysis
```

## Architecture

Thin pipeline: **fetch → model → render → write**

```
main.go
  └─ loads .env, dispatches subcommand
       ├─ internal/auth/auth.go      OAuth2 flow + token lifecycle
       ├─ internal/client/client.go  Authenticated HTTP GET, 429 retry
       ├─ internal/fetch/fetch.go    Paginated API calls → DayData aggregate
       ├─ internal/models/models.go  WHOOP v2 JSON structs + SPORT_NAMES map
       └─ internal/render/render.go  text/template rendering + helpers
```

**Data flow for `daily`:**
1. `auth.RefreshIfNeeded()` — load/refresh `tokens.json`
2. `client.NewClient(token)` — wrap token
3. `fetch.GetDayData(c, date)` — concurrent goroutines fetch cycle, recovery, sleeps, workouts
4. `render.RenderDaily(data, tmplPath)` — execute `templates/daily.md.tmpl`
5. Write file to output directory

## Templates

Templates live in `templates/` and are loaded from disk at runtime. Override
location with `WHOOP_TEMPLATES_DIR` env var.

Available template helpers: `millisToMinutes`, `recoveryColor`, `strainCategory`,
`sportName`, `primarySleep`

## Output Files

| Command | Filename |
|---|---|
| `daily` | `daily-2026-02-20.md` |
| `weekly` | `weekly-2026-W08.md` |

## Notes

- `fetch-all` skips dates with no WHOOP cycle data and throttles 500ms between
  days to stay within API rate limits
- `persona` outputs to stdout — pipe or copy into your health persona note
- `tokens.json` and `.env` are gitignored; never commit them
