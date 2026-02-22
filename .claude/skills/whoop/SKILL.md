---
name: whoop
description: Pull WHOOP fitness data and generate health summary pages for the digital garden / Obsidian vault. Use this skill whenever the user asks about their WHOOP data, recovery scores, HRV, sleep analysis, strain, workouts, or wants to generate daily/weekly health notes, update their health persona file, or analyze fitness trends. Also trigger when the user says things like "how did I sleep", "what's my recovery", "update my health page", "run whoop daily", or "generate my weekly summary". This skill operates the whoop-garden Go CLI to fetch data and render markdown.
---

# WHOOP Digital Garden Skill

Fetches data from the WHOOP v2 API via the `whoop-garden` Go CLI and generates
structured markdown files for the Obsidian digital garden.

**Project directory:** Set `$WHOOP_GARDEN_DIR` in your environment, or `cd` to the project root before running commands. All CLI commands must be run from the project directory.

## Quick Reference

```bash
cd "$WHOOP_GARDEN_DIR"

go run . auth                        # one-time OAuth (or after token expiry)
go run . daily                       # today's daily note
go run . daily --date 2026-02-19     # specific date
go run . weekly                      # this week's summary
go run . weekly --date 2026-02-17    # week containing that date
go run . persona                     # 30d persona section → stdout
go run . persona --days 60           # custom window
go run . fetch-all --days 30         # backfill N daily notes
```

**Output location:** `$OBSIDIAN_VAULT_PATH/Health/WHOOP/` if that env var is set in `.env`,
otherwise `./output/` relative to project root. Set `OBSIDIAN_VAULT_PATH` in `.env` to
route files directly to the vault.

**Credentials:** All secrets in `.env` at project root — never log or share.

---

## CLI Command Reference

| Command | What it does | Output |
|---|---|---|
| `auth` | OAuth2 flow, saves `tokens.json` | `tokens.json` |
| `daily` | Fetch today, render daily note | `daily-YYYY-MM-DD.md` |
| `daily --date YYYY-MM-DD` | Specific date | `daily-YYYY-MM-DD.md` |
| `weekly` | Mon–Sun of current week | `weekly-YYYY-W##.md` (e.g. `weekly-2026-W08.md`) |
| `weekly --date YYYY-MM-DD` | Week containing that date | `weekly-YYYY-W##.md` |
| `persona [--days N]` | Rolling N-day persona section | **stdout only** — pipe or copy manually |
| `fetch-all [--days N]` | Write N daily notes (500ms throttle, skips no-data days) | one file per day |

---

## Output File Formats

### Daily Note (`daily-YYYY-MM-DD.md`)

Frontmatter: `date` + `tags: [whoop, daily-health, YYYY]`

Sections rendered:
- **Summary callout** — one-line recovery % + strain
- **Recovery table** — Score, HRV, RHR, SpO₂, Skin Temp
- **Sleep table(s)** — In Bed, Awake, Light, SWS (Deep), REM, Performance %, Efficiency %, Respiratory Rate, Disturbances. Naps rendered separately.
- **Strain table** — Day Strain (with category label), Avg HR, Max HR, Calories (kJ)
- **Workouts table** — one section per workout: Strain, Avg HR, Max HR, Calories, Distance (if >0)

### Weekly Note (`weekly-YYYY-W##.md`)

Sections rendered:
- Aggregate Stats table (Avg Recovery, HRV, RHR, Strain, Sleep, Total Workouts)
- Recovery Distribution (🟢/🟡/🔴 day counts)
- Daily Breakdown table (one row per day)
- Workouts This Week table
- Highlights — best recovery day, worst recovery day (with scores)

### Persona Section (stdout)

```markdown
## Health Persona (30-Day Rolling Summary)

**Period:** 2026-01-21 → 2026-02-19

### Recovery
- Average Recovery Score: **65%**
- Average HRV: **43.9 ms**
- HRV Trend: **Stable**  (or "Improving (+0.8%/day)" / "Declining (-1.2%/day)")
- Average RHR: **55 bpm**

### Sleep
- Average Sleep Duration: **7h 17m**
- Average Sleep Performance: **76%**

### Strain
- Average Day Strain: **10.5**
- Total Workouts: **43**

### Recovery Distribution
- Green (67–100): 12 days
- Yellow (34–66): 15 days
- Red (0–33): 2 days
```

---

## Data Available from WHOOP API

### Recovery
- **Recovery Score** (0–100%) — composite readiness
  - 🟢 Green: 67–100% — ready to push hard
  - 🟡 Yellow: 34–66% — moderate load appropriate
  - 🔴 Red: 0–33% — prioritize rest
- **HRV (RMSSD ms)** — heaviest weighted input (~65–70% of score). Compare to user's 30d personal baseline, not population averages.
- **Resting Heart Rate (bpm)** — measured during deep sleep
- **SpO₂ (%)** — blood oxygen; below 95% is notable
- **Skin Temperature (°C)** — raw value from sensor (not delta in API)

### Sleep
- **Sleep stages (in milliseconds → rendered as Xh Ym):**
  - SWS / Deep — physical repair, HGH release
  - REM — cognitive processing, emotional regulation
  - Light — transitional
  - Awake time during night
- **Respiratory Rate (breaths/min)** — tracked in sleep section. Baseline is 13–18. Spikes of 1–2 above baseline can precede illness by 24–48h — flag proactively.
- **Sleep Performance %** — actual sleep vs sleep need
- **Sleep Efficiency %** — time asleep / time in bed
- **Disturbance Count**

### Strain
Strain uses a non-linear Borg Scale (0–21). Categories from the code:
- 0–6: **Minimal**
- 7–9: **Light** — active recovery territory
- 10–13: **Moderate** — maintains fitness
- 14–17: **Strenuous** — builds fitness
- 18–21: **All Out** — significant recovery needed next day

### Workouts
- Sport type, duration (start/end timestamps), workout strain, avg/max HR, calories, distance

### What the API Does NOT Expose
Not available via API — do not attempt to fetch:
- Raw HR time-series (minute-by-minute)
- Stress Monitor score
- VO₂ Max
- Healthspan / WHOOP Age
- Blood Pressure, ECG data

---

## Data Interpretation Guide

### HRV Context
HRV is personal and baseline-normalized — never compare absolute ms values to population averages.
- HRV trending up over 2–3 weeks → positive fitness adaptation
- HRV sudden drop (>15ms below baseline in one night) → likely acute stressor (alcohol, illness, hard effort)
- HRV chronically suppressed → overtraining or chronic stress
- Alcohol is one of the largest HRV suppressors — even moderate consumption the night before matters

### Recovery Score Nuance
Measures cardiovascular readiness, not muscle soreness or mental fatigue. A Green score
with heavy DOMS is normal. Trust it directionally, not as gospel.
HRV weights ~65–70%, RHR ~20%, Respiratory Rate ~10–15%, all normalized to 30d baseline.

### Strain-Recovery Relationship
Prior-day strain is the strongest predictor of next-day recovery. Strain 15+ typically
results in sub-56% recovery the following day. Look for strain peaks followed by recovery
days before the next hard effort.

### Sleep Architecture Priorities
When summarizing, prioritize in this order:
1. SWS / Deep Sleep — adults need 15–25% of total sleep
2. REM — should be 20–25% of total sleep
3. Total duration vs need
4. Disturbance count
5. Consistency score (long-term predictor)

### Respiratory Rate as Illness Canary
A sustained rise of 1–2 breaths/min above personal baseline often precedes illness
by 24–48 hours. Flag this proactively in summaries.

---

## Workflow: Answering WHOOP Questions

When the user asks about their health data:

1. **Check vault first** — look in `$OBSIDIAN_VAULT_PATH/Health/WHOOP/` for recent daily notes
2. **If data is stale or missing** → run the appropriate CLI command
3. **Read the generated files** to get actual numbers
4. **Apply interpretation above** to give context beyond raw scores
5. **Surface patterns** — check 3–7 days of notes for trends, not just today

Never make up or estimate WHOOP scores. Always fetch real data or read from vault files.

---

## Common Scenarios

### "Update my health page / run daily"
```bash
cd "$WHOOP_GARDEN_DIR" && go run . daily
```
Read the output file and summarize key insights.

### "How was my recovery this week"
```bash
cd "$WHOOP_GARDEN_DIR" && go run . weekly
```
Report avg recovery, HRV trend, best/worst day, notable patterns.

### "Update my persona"
```bash
cd "$WHOOP_GARDEN_DIR" && go run . persona
```
Output goes to stdout. Ask user if they want it written to their persona note or review first.

### "Backfill my vault"
```bash
cd "$WHOOP_GARDEN_DIR" && go run . fetch-all --days 90
```
Generates up to 90 daily notes with a 500ms throttle between days. Skips dates with
no WHOOP cycle data (e.g., days before the user got their device).

### "Why is my recovery low"
Read the last 3–5 daily notes. Look for:
- Elevated strain the prior day(s)
- Low deep sleep % or high disturbance count
- HRV below 30d baseline
- Elevated respiratory rate or skin temp
Report specific contributing factors, not just "your score was low".

### Auth errors / token expired
```bash
cd "$WHOOP_GARDEN_DIR" && go run . auth
```
Browser opens to WHOOP's auth page. After approving, callback is captured on :3000
and new tokens saved to `tokens.json`.

---

## Environment Variables

All loaded from `.env` at project root.

| Variable | Required | Description |
|---|---|---|
| `WHOOP_CLIENT_ID` | Yes | From developer.whoop.com |
| `WHOOP_CLIENT_SECRET` | Yes | Keep secret — never log |
| `WHOOP_REDIRECT_URI` | Yes | Must be `http://localhost:3000/callback` |
| `OBSIDIAN_VAULT_PATH` | Recommended | Absolute path to vault root (no trailing slash) |
| `WHOOP_TEMPLATES_DIR` | Optional | Override template directory |

If `OBSIDIAN_VAULT_PATH` is not set, output goes to `./output/` in the project directory.

---

## Troubleshooting

**"tokens.json not found"** → Run `go run . auth`

**"401 Unauthorized"** → Token expired. Run `go run . auth`

**"429 Too Many Requests"** → Client auto-retries once with backoff. If it persists,
wait 60 seconds. WHOOP limits are generous for personal use but paginated backfills can hit them.

**No data for a date** → WHOOP only scores after sleep completes. Fetching today before
morning sync returns incomplete data — use `--date yesterday` or wait until after sync.

**Wrong output location** → Check `OBSIDIAN_VAULT_PATH` in `.env`. CLI creates
`Health/WHOOP/` subdirectory automatically if it doesn't exist.
