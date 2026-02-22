# Commands

All commands auto-load `.env` from the current working directory and
auto-refresh OAuth tokens if they are expiring within 5 minutes.

---

## auth

```bash
go run . auth
```

Runs the full OAuth 2.0 authorization code flow:

1. Opens `https://api.prod.whoop.com/oauth/oauth2/auth` in your default
   browser (macOS `open`)
2. Starts a local HTTP server on `:3000` to receive the callback
3. Exchanges the authorization code for access and refresh tokens
4. Saves tokens to `tokens.json` in the current directory

If the browser does not open automatically, the full auth URL is printed to
stdout — copy and paste it manually.

Tokens auto-refresh on subsequent commands. You should only need to run `auth`
once, or if `tokens.json` is deleted or the refresh token expires.

**Requires:** `WHOOP_CLIENT_ID`, `WHOOP_CLIENT_SECRET`, `WHOOP_REDIRECT_URI`
in `.env`. Port `3000` must be free.

---

## daily

```bash
go run . daily [--date YYYY-MM-DD]
```

Generates a daily markdown note for the given date (default: today).

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--date` | today | Date in `YYYY-MM-DD` format |

**Output:** `<output>/<year>/daily-YYYY-MM-DD.md`

**What it fetches:** cycle, recovery, sleeps (including naps), workouts. If
WHOOP has no cycle for the requested date, the file is still written with
empty sections.

---

## weekly

```bash
go run . weekly [--date YYYY-MM-DD]
```

Generates a weekly summary note for the ISO week (Mon–Sun) containing the
given date (default: the current week).

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--date` | today | Any date within the target week |

**Output:** `<output>/<year>/weekly-YYYY-Www.md`
Example: `weekly-2026-W08.md`

**What it fetches:** calls `daily` data for each of the 7 days, aggregates
into weekly averages, recovery distribution, and best/worst day highlights.
Future days within the week are included as empty placeholders so the note
can be partially generated mid-week.

The year in the filename is the ISO year (which differs from the calendar year
near year boundaries — e.g. Dec 31 may belong to week 1 of the following year).

---

## persona

```bash
go run . persona [--days N]
```

Generates a rolling N-day health persona section with aggregated stats.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--days` | 30 | Number of days to include |

**Output:**
- If `OBSIDIAN_VAULT_PATH` is set: writes to
  `<vault>/01-ai-brain/context-packs/WHOOP Health Persona.md`
- Otherwise: prints to stdout

**Contents:** average recovery score, HRV with linear regression trend label
(Improving / Declining / Stable), RHR, sleep duration and performance,
average strain, workout count, and green/yellow/red day distribution.

The HRV trend is computed as a least-squares slope over the N days, normalized
by mean HRV to produce a daily percentage change. Values above +0.5%/day are
labelled "Improving", below −0.5%/day "Declining", otherwise "Stable".

---

## fetch-all

```bash
go run . fetch-all [--days N]
```

Fetches and writes daily notes for the last N days, overwriting any existing
files.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--days` | 30 | Number of days to backfill |

Sleeps 500 ms between each day's API calls to respect rate limits. Days with
no WHOOP cycle data are skipped (noted as "Skipped: no data").

Use `catch-up` instead of `fetch-all` if you only want to fill gaps without
overwriting notes you have already edited.

---

## catch-up

```bash
go run . catch-up [--days N]
```

Scans the output directory for missing daily notes and fetches only those
dates. Existing files are never overwritten.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--days` | 30 | Number of days to scan |

**Behaviour:**
1. Scans `<output>/<year>/daily-YYYY-MM-DD.md` for each day in the window
2. If all files exist, exits immediately with "All caught up"
3. Only authenticates and calls the API if missing files are found
4. Sleeps 500 ms between API calls

This is the preferred command for scheduled runs (launchd, cron) since it
is a no-op when everything is up to date and avoids unnecessary API calls.

---

## Output Directory

Files are written to:

```
$OBSIDIAN_VAULT_PATH/Health/WHOOP/<year>/   # if vault path is set
./output/<year>/                             # fallback
```

The year subdirectory is created automatically.
