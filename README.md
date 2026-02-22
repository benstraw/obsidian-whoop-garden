# whoop-garden

Pulls data from the [WHOOP v2 API](https://developer.whoop.com) and renders
structured Obsidian markdown notes — daily summaries, weekly rollups, and a
rolling 30-day health persona section.

No external dependencies. stdlib only.

---

## Quick Start

### 1. Register a WHOOP app

Go to [developer.whoop.com](https://developer.whoop.com) and create an app —
it's free and takes about two minutes. Set the redirect URI to
`http://localhost:3000/callback`.

### 2. Configure environment

```bash
cp .env.example .env
```

```
WHOOP_CLIENT_ID=your_client_id
WHOOP_CLIENT_SECRET=your_client_secret
WHOOP_REDIRECT_URI=http://localhost:3000/callback

# Optional
OBSIDIAN_VAULT_PATH=/path/to/your/vault
```

### 3. Authenticate

```bash
go run . auth
```

Opens a browser to WHOOP's OAuth page. Tokens are saved to `tokens.json` and
auto-refresh — you should only need to do this once.

### 4. Generate notes

```bash
go run . daily                       # today's note
go run . daily --date 2026-02-19     # specific date
go run . weekly                      # this week's summary
go run . catch-up --days 30          # backfill only missing notes
go run . persona                     # 30-day rolling health summary
```

---

## Build & Test

```bash
go build ./...                        # compile
go build -o bin/whoop-garden .        # produce binary
go vet ./...                          # static analysis
go test ./...                         # run all tests
go test ./... -short                  # skip slow tests (~3s sleep)
```

---

## Output

Files are written to `$OBSIDIAN_VAULT_PATH/Health/WHOOP/<year>/` when the
vault path is set, otherwise `./output/<year>/`.

| Command | Filename |
|---|---|
| `daily` | `daily-2026-02-20.md` |
| `weekly` | `weekly-2026-W08.md` |
| `persona` | `<vault>/01-ai-brain/context-packs/WHOOP Health Persona.md` |

---

## Documentation

| Doc | Contents |
|---|---|
| [docs/commands.md](docs/commands.md) | All commands, flags, and behaviour details |
| [docs/architecture.md](docs/architecture.md) | Package map, data flow, design decisions |
| [docs/templates.md](docs/templates.md) | Template system, all helpers, data structures, customisation |
| [docs/auth-flow.md](docs/auth-flow.md) | OAuth flow, token storage, refresh, troubleshooting |
| [docs/testing.md](docs/testing.md) | Running tests, coverage by package, known gaps, adding new tests |

---

## Notes

- `tokens.json` and `.env` are gitignored — never commit them
- `catch-up` skips existing files; `fetch-all` overwrites them
- Port `3000` must be free when running `auth`
