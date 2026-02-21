package render

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"text/template"
	"time"

	"github.com/benstraw/whoop-garden/internal/fetch"
	"github.com/benstraw/whoop-garden/internal/models"
)

const personaTemplate = `---
type: context
tags: [ai-brain/context, fitness/whoop]
updated: {{.GeneratedDate}}
---

# WHOOP Health Persona

> [!info] Auto-generated
> Regenerate with ` + "`" + `whoop-garden persona` + "`" + `. Covers {{.PeriodStart}} → {{.PeriodEnd}}.

## Health Persona (30-Day Rolling Summary)

**Period:** {{.PeriodStart}} → {{.PeriodEnd}}

### Recovery
- Average Recovery Score: **{{printf "%.0f" .AvgRecovery}}%**
- Average HRV: **{{printf "%.1f" .AvgHRV}} ms**
- HRV Trend: **{{.HRVTrend}}**
- Average RHR: **{{printf "%.0f" .AvgRHR}} bpm**

### Sleep
- Average Sleep Duration: **{{millisToMinutes .AvgSleepMillis}}**
- Average Sleep Performance: **{{printf "%.0f" .AvgSleepPerf}}%**

### Strain
- Average Day Strain: **{{printf "%.1f" .AvgStrain}}**
- Total Workouts: **{{.TotalWorkouts}}**

### Recovery Distribution
- Green (67–100): {{.GreenDays}} days
- Yellow (34–66): {{.YellowDays}} days
- Red (0–33): {{.RedDays}} days
`

// FuncMap returns the template helper functions.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"millisToMinutes": MillisToMinutes,
		"recoveryColor":   RecoveryColor,
		"strainCategory":  StrainCategory,
		"sportName":       SportName,
		"primarySleep":    PrimarySleep,
		"nonNapSleeps":    NonNapSleeps,
		"prevDay":         PrevDay,
		"nextDay":         NextDay,
		"isoWeek":         ISOWeekStr,
		"prevWeek":        PrevWeekStr,
		"nextWeek":        NextWeekStr,
		"prevDayYear":     PrevDayYear,
		"nextDayYear":     NextDayYear,
		"isoWeekYear":     ISOWeekYear,
		"prevWeekYear":    PrevWeekYear,
		"nextWeekYear":    NextWeekYear,
	}
}

// PrevDayYear returns the calendar year of the day before t.
func PrevDayYear(t time.Time) int { return t.AddDate(0, 0, -1).Year() }

// NextDayYear returns the calendar year of the day after t.
func NextDayYear(t time.Time) int { return t.AddDate(0, 0, 1).Year() }

// ISOWeekYear returns the ISO year for the week containing t.
func ISOWeekYear(t time.Time) int { year, _ := t.ISOWeek(); return year }

// PrevWeekYear returns the ISO year for the week before t.
func PrevWeekYear(t time.Time) int { year, _ := t.AddDate(0, 0, -7).ISOWeek(); return year }

// NextWeekYear returns the ISO year for the week after t.
func NextWeekYear(t time.Time) int { year, _ := t.AddDate(0, 0, 7).ISOWeek(); return year }

// PrimarySleep returns the longest non-nap sleep from a slice, or nil if none.
func PrimarySleep(sleeps []models.Sleep) *models.Sleep {
	var best *models.Sleep
	for i := range sleeps {
		s := &sleeps[i]
		if s.Nap {
			continue
		}
		if best == nil || s.Score.StageSummary.TotalInBedTimeMilli > best.Score.StageSummary.TotalInBedTimeMilli {
			best = s
		}
	}
	return best
}

// IndexedSleep wraps a Sleep with its ordinal position among non-nap sleeps.
type IndexedSleep struct {
	Index int
	Sleep models.Sleep
}

// NonNapSleeps filters to non-nap entries and attaches ordinal index.
func NonNapSleeps(sleeps []models.Sleep) []IndexedSleep {
	var result []IndexedSleep
	for _, s := range sleeps {
		if !s.Nap {
			result = append(result, IndexedSleep{Index: len(result), Sleep: s})
		}
	}
	return result
}

// PrevDay returns "YYYY-MM-DD" for the day before t.
func PrevDay(t time.Time) string { return t.AddDate(0, 0, -1).Format("2006-01-02") }

// NextDay returns "YYYY-MM-DD" for the day after t.
func NextDay(t time.Time) string { return t.AddDate(0, 0, 1).Format("2006-01-02") }

// ISOWeekStr returns "YYYY-Www" for the ISO week containing t.
func ISOWeekStr(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// PrevWeekStr returns "YYYY-Www" for the week before t.
func PrevWeekStr(t time.Time) string { return ISOWeekStr(t.AddDate(0, 0, -7)) }

// NextWeekStr returns "YYYY-Www" for the week after t.
func NextWeekStr(t time.Time) string { return ISOWeekStr(t.AddDate(0, 0, 7)) }

// MillisToMinutes converts milliseconds to a "Xh Ym" string.
func MillisToMinutes(ms int64) string {
	total := ms / 1000 / 60
	h := total / 60
	m := total % 60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

// RecoveryColor returns "green", "yellow", or "red" based on score.
func RecoveryColor(score float64) string {
	switch {
	case score >= 67:
		return "green"
	case score >= 34:
		return "yellow"
	default:
		return "red"
	}
}

// StrainCategory returns a label for a strain value.
func StrainCategory(strain float64) string {
	switch {
	case strain >= 18:
		return "All Out"
	case strain >= 14:
		return "Strenuous"
	case strain >= 10:
		return "Moderate"
	case strain >= 7:
		return "Light"
	default:
		return "Minimal"
	}
}

// SportName returns the human-readable name for a WHOOP sport ID.
func SportName(id int) string {
	if name, ok := models.SPORT_NAMES[id]; ok {
		return name
	}
	return fmt.Sprintf("Sport(%d)", id)
}

// RenderDaily renders a daily markdown note from a file template.
func RenderDaily(data fetch.DayData, tmplPath string) (string, error) {
	tmpl, err := template.New("daily").Funcs(FuncMap()).ParseFiles(tmplPath)
	if err != nil {
		return "", fmt.Errorf("parse daily template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "daily.md.tmpl", data); err != nil {
		return "", fmt.Errorf("render daily template: %w", err)
	}
	return buf.String(), nil
}

// RenderWeekly renders a weekly markdown note from a file template.
func RenderWeekly(data []fetch.DayData, tmplPath string) (string, error) {
	tmpl, err := template.New("weekly").Funcs(FuncMap()).ParseFiles(tmplPath)
	if err != nil {
		return "", fmt.Errorf("parse weekly template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "weekly.md.tmpl", data); err != nil {
		return "", fmt.Errorf("render weekly template: %w", err)
	}
	return buf.String(), nil
}

// personaData holds aggregated stats for the persona template.
type personaData struct {
	GeneratedDate  string
	PeriodStart    string
	PeriodEnd      string
	AvgRecovery    float64
	AvgHRV         float64
	HRVTrend       string
	AvgRHR         float64
	AvgSleepMillis int64
	AvgSleepPerf   float64
	AvgStrain      float64
	TotalWorkouts  int
	GreenDays      int
	YellowDays     int
	RedDays        int
}

// RenderPersonaSection generates a markdown persona section using 30d rolling data.
func RenderPersonaSection(data []fetch.DayData) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("no data provided for persona")
	}

	pd := aggregatePersonaData(data)

	funcMap := FuncMap()
	// millisToMinutes is used in template directly via funcMap
	tmpl, err := template.New("persona").Funcs(funcMap).Parse(personaTemplate)
	if err != nil {
		return "", fmt.Errorf("parse persona template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, pd); err != nil {
		return "", fmt.Errorf("render persona template: %w", err)
	}
	return buf.String(), nil
}

func aggregatePersonaData(data []fetch.DayData) personaData {
	var (
		totalRecovery    float64
		totalHRV         float64
		totalRHR         float64
		totalSleepMillis int64
		totalSleepPerf   float64
		totalStrain      float64
		totalWorkouts    int
		greenDays        int
		yellowDays       int
		redDays          int
		recoveryCount    int
		sleepCount       int
		cycleCount       int
		hrvValues        []float64
	)

	for _, d := range data {
		if d.Recovery != nil && d.Recovery.ScoreState == "SCORED" {
			totalRecovery += d.Recovery.Score.RecoveryScore
			totalHRV += d.Recovery.Score.HrvRmssdMilli
			totalRHR += d.Recovery.Score.RestingHeartRate
			hrvValues = append(hrvValues, d.Recovery.Score.HrvRmssdMilli)
			recoveryCount++

			switch RecoveryColor(d.Recovery.Score.RecoveryScore) {
			case "green":
				greenDays++
			case "yellow":
				yellowDays++
			case "red":
				redDays++
			}
		}

		for _, s := range d.Sleeps {
			if !s.Nap && s.ScoreState == "SCORED" {
				totalSleepMillis += s.Score.StageSummary.TotalInBedTimeMilli
				totalSleepPerf += s.Score.SleepPerformance
				sleepCount++
			}
		}

		if d.Cycle != nil && d.Cycle.ScoreState == "SCORED" {
			totalStrain += d.Cycle.Score.Strain
			cycleCount++
		}

		totalWorkouts += len(d.Workouts)
	}

	avg := func(total float64, count int) float64 {
		if count == 0 {
			return 0
		}
		return total / float64(count)
	}
	avgI := func(total int64, count int) int64 {
		if count == 0 {
			return 0
		}
		return total / int64(count)
	}

	first := data[0].Date.Format("2006-01-02")
	last := data[len(data)-1].Date.Format("2006-01-02")

	return personaData{
		GeneratedDate:  time.Now().Format("2006-01-02"),
		PeriodStart:    first,
		PeriodEnd:      last,
		AvgRecovery:    avg(totalRecovery, recoveryCount),
		AvgHRV:         avg(totalHRV, recoveryCount),
		HRVTrend:       hrvTrendLabel(hrvValues),
		AvgRHR:         avg(totalRHR, recoveryCount),
		AvgSleepMillis: avgI(totalSleepMillis, sleepCount),
		AvgSleepPerf:   avg(totalSleepPerf, sleepCount),
		AvgStrain:      avg(totalStrain, cycleCount),
		TotalWorkouts:  totalWorkouts,
		GreenDays:      greenDays,
		YellowDays:     yellowDays,
		RedDays:        redDays,
	}
}

// hrvTrendLabel computes a linear regression slope over HRV values and returns a label.
func hrvTrendLabel(vals []float64) string {
	n := len(vals)
	if n < 3 {
		return "Insufficient data"
	}

	// Least-squares slope: slope = (n*Σ(xy) - Σx*Σy) / (n*Σx² - (Σx)²)
	var sumX, sumY, sumXY, sumX2 float64
	for i, y := range vals {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	fn := float64(n)
	denom := fn*sumX2 - sumX*sumX
	if denom == 0 {
		return "Stable"
	}
	slope := (fn*sumXY - sumX*sumY) / denom

	// Normalize by mean HRV to get percentage change per day.
	meanHRV := sumY / fn
	if meanHRV == 0 {
		return "Stable"
	}
	normalizedSlope := slope / meanHRV * 100

	switch {
	case normalizedSlope > 0.5:
		return fmt.Sprintf("Improving (+%.1f%%/day)", math.Abs(normalizedSlope))
	case normalizedSlope < -0.5:
		return fmt.Sprintf("Declining (%.1f%%/day)", normalizedSlope)
	default:
		return "Stable"
	}
}

// WeekStats aggregates weekly data for the weekly template.
type WeekStats struct {
	Days          []fetch.DayData
	WeekStart     string
	WeekEnd       string
	AvgRecovery   float64
	AvgHRV        float64
	AvgRHR        float64
	AvgStrain     float64
	AvgSleepStr   string
	GreenDays     int
	YellowDays    int
	RedDays       int
	TotalWorkouts int
	BestDay       *fetch.DayData
	WorstDay      *fetch.DayData
}

// BuildWeekStats aggregates a slice of DayData into WeekStats for templates.
func BuildWeekStats(days []fetch.DayData) WeekStats {
	ws := WeekStats{Days: days}
	if len(days) == 0 {
		return ws
	}
	ws.WeekStart = days[0].Date.Format("2006-01-02")
	ws.WeekEnd = days[len(days)-1].Date.Format("2006-01-02")

	var totalRec, totalHRV, totalRHR, totalStrain float64
	var totalSleepMs int64
	var recCount, sleepCount, strainCount int
	var bestScore, worstScore float64
	bestScore = -1
	worstScore = 101

	for i, d := range days {
		ws.TotalWorkouts += len(d.Workouts)
		if d.Recovery != nil && d.Recovery.ScoreState == "SCORED" {
			s := d.Recovery.Score.RecoveryScore
			totalRec += s
			totalHRV += d.Recovery.Score.HrvRmssdMilli
			totalRHR += d.Recovery.Score.RestingHeartRate
			recCount++

			switch RecoveryColor(s) {
			case "green":
				ws.GreenDays++
			case "yellow":
				ws.YellowDays++
			case "red":
				ws.RedDays++
			}

			if s > bestScore {
				bestScore = s
				cp := days[i]
				ws.BestDay = &cp
			}
			if s < worstScore {
				worstScore = s
				cp := days[i]
				ws.WorstDay = &cp
			}
		}

		if d.Cycle != nil && d.Cycle.ScoreState == "SCORED" {
			totalStrain += d.Cycle.Score.Strain
			strainCount++
		}

		for _, sl := range d.Sleeps {
			if !sl.Nap && sl.ScoreState == "SCORED" {
				totalSleepMs += sl.Score.StageSummary.TotalInBedTimeMilli
				sleepCount++
			}
		}
	}

	avg := func(t float64, c int) float64 {
		if c == 0 {
			return 0
		}
		return t / float64(c)
	}
	ws.AvgRecovery = avg(totalRec, recCount)
	ws.AvgHRV = avg(totalHRV, recCount)
	ws.AvgRHR = avg(totalRHR, recCount)
	ws.AvgStrain = avg(totalStrain, strainCount)
	var avgSleepMs int64
	if sleepCount > 0 {
		avgSleepMs = totalSleepMs / int64(sleepCount)
	}
	ws.AvgSleepStr = MillisToMinutes(avgSleepMs)

	return ws
}

// weeklyTemplateData is passed to the weekly template.
type weeklyTemplateData struct {
	Stats WeekStats
}

// RenderWeeklyFromStats renders a weekly note from pre-aggregated WeekStats.
func RenderWeeklyFromStats(stats WeekStats, tmplPath string) (string, error) {
	funcMap := FuncMap()
	funcMap["join"] = strings.Join
	tmpl, err := template.New("weekly.md.tmpl").Funcs(funcMap).ParseFiles(tmplPath)
	if err != nil {
		return "", fmt.Errorf("parse weekly template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "weekly.md.tmpl", weeklyTemplateData{Stats: stats}); err != nil {
		return "", fmt.Errorf("render weekly template: %w", err)
	}
	return buf.String(), nil
}
