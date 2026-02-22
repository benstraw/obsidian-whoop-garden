# Templates

Templates are standard Go `text/template` files with an Obsidian-flavored
markdown output. They live in `templates/` and are loaded from disk at runtime,
so you can edit them without recompiling.

## Template Resolution

The templates directory is resolved in order:

1. `$WHOOP_TEMPLATES_DIR` — set this env var to use templates from any path
2. `./templates/` relative to the current working directory (used by `go run .`)
3. `<binary_dir>/templates/` next to the compiled binary

## Available Templates

| File | Command | Data type passed |
|------|---------|-----------------|
| `daily.md.tmpl` | `daily`, `fetch-all`, `catch-up` | `fetch.DayData` |
| `weekly.md.tmpl` | `weekly` | `render.WeekStats` (wrapped in `weeklyTemplateData`) |

The `persona` output uses a compiled-in template string in `render/render.go`
and is not a file on disk.

## Template Helpers (FuncMap)

All helpers are available in both `daily.md.tmpl` and `weekly.md.tmpl`.

### `millisToMinutes`

Converts milliseconds to a human-readable duration string.

```
{{ millisToMinutes 28800000 }}   → "8h 0m"
{{ millisToMinutes 5400000 }}    → "1h 30m"
{{ millisToMinutes 60000 }}      → "1m"
```

### `recoveryColor`

Returns a color label based on WHOOP's recovery zones.

```
{{ recoveryColor 80 }}   → "green"   (67–100)
{{ recoveryColor 50 }}   → "yellow"  (34–66)
{{ recoveryColor 20 }}   → "red"     (0–33)
```

Useful for Obsidian callout types: `> [!{{ recoveryColor .Score }}]`

### `strainCategory`

Returns a strain label.

```
{{ strainCategory 19 }}   → "All Out"     (18+)
{{ strainCategory 15 }}   → "Strenuous"   (14–17)
{{ strainCategory 11 }}   → "Moderate"    (10–13)
{{ strainCategory 8 }}    → "Light"       (7–9)
{{ strainCategory 3 }}    → "Minimal"     (<7)
```

### `sportName`

Returns the human-readable name for a WHOOP sport ID. Falls back to
`"Sport(N)"` for unknown IDs.

```
{{ sportName 0 }}     → "Running"
{{ sportName 44 }}    → "Yoga"
{{ sportName 100 }}   → "Jiu Jitsu"
{{ sportName 999 }}   → "Sport(999)"
```

### `primarySleep`

Returns a pointer to the longest non-nap sleep from a slice, or nil.

```
{{ with primarySleep .Sleeps }}
  Duration: {{ millisToMinutes .Score.StageSummary.TotalInBedTimeMilli }}
{{ end }}
```

### `nonNapSleeps`

Filters a sleep slice to non-nap entries and returns `[]IndexedSleep`, each
with an `Index` (0-based ordinal) and `Sleep` field.

```
{{ range nonNapSleeps .Sleeps }}
  Sleep {{ .Index }}: {{ millisToMinutes .Sleep.Score.StageSummary.TotalInBedTimeMilli }}
{{ end }}
```

### Date Navigation

```
{{ prevDay .Date }}          → "2026-02-09"  (YYYY-MM-DD)
{{ nextDay .Date }}          → "2026-02-11"
{{ isoWeek .Date }}          → "2026-W07"
{{ prevWeek .Date }}         → "2026-W06"
{{ nextWeek .Date }}         → "2026-W08"
{{ prevDayYear .Date }}      → 2025  (int, for year subdir in wikilinks)
{{ nextDayYear .Date }}      → 2026
{{ isoWeekYear .Date }}      → 2026  (ISO year, may differ from calendar year)
{{ prevWeekYear .Date }}     → 2025
{{ nextWeekYear .Date }}     → 2026
```

Year helpers exist because Obsidian wikilinks include the year subdirectory:
`[[2026/daily-2026-02-10]]`. Near year/ISO-week boundaries the year can differ
from the date's calendar year.

## Data Structures

### `DayData` (daily template)

```go
type DayData struct {
    Date     time.Time
    Cycle    *models.Cycle    // nil if no cycle found
    Recovery *models.Recovery // nil if no recovery found
    Sleeps   []models.Sleep
    Workouts []models.Workout
}
```

Always check for nil before accessing Cycle or Recovery:

```
{{ if .Cycle }}
  Strain: {{ printf "%.1f" .Cycle.Score.Strain }}
{{ else }}
  No cycle data
{{ end }}
```

### `WeekStats` (weekly template, accessed as `.Stats`)

```go
type WeekStats struct {
    Days          []fetch.DayData
    WeekStart     string   // "YYYY-MM-DD"
    WeekEnd       string
    AvgRecovery   float64
    AvgHRV        float64
    AvgRHR        float64
    AvgStrain     float64
    AvgSleepStr   string   // pre-formatted, e.g. "7h 30m"
    GreenDays     int
    YellowDays    int
    RedDays       int
    TotalWorkouts int
    BestDay       *fetch.DayData
    WorstDay      *fetch.DayData
}
```

Access in template as `.Stats.AvgRecovery`, `.Stats.Days`, etc.

## Customising Templates

1. Copy the template you want to change
2. Set `WHOOP_TEMPLATES_DIR` to the directory containing your copy
3. Run any command — changes take effect immediately, no rebuild needed

You can add new FuncMap helpers by editing `render/render.go:FuncMap()` and
rebuilding.

## Score State

WHOOP scores are not always available immediately. The `ScoreState` field on
Cycle, Recovery, Sleep, and Workout can be:

- `"SCORED"` — score is available and valid
- `"PENDING_SCORE"` — processing in progress
- `"UNSCORABLE"` — insufficient data

Templates should guard against non-SCORED states when displaying numeric
scores:

```
{{ if eq .Recovery.ScoreState "SCORED" }}
  HRV: {{ printf "%.1f" .Recovery.Score.HrvRmssdMilli }} ms
{{ else }}
  Pending
{{ end }}
```
