package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/benstraw/whoop-garden/internal/fetch"
	"github.com/benstraw/whoop-garden/internal/models"
)

// --- MillisToMinutes ---

func TestMillisToMinutes(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{0, "0m"},
		{60_000, "1m"},
		{3_600_000, "1h 0m"},
		{5_400_000, "1h 30m"},
		{28_800_000, "8h 0m"},
		{30_600_000, "8h 30m"},
		{45_000, "0m"}, // 45 seconds → integer truncation to 0m
	}
	for _, tc := range tests {
		got := MillisToMinutes(tc.ms)
		if got != tc.want {
			t.Errorf("MillisToMinutes(%d) = %q, want %q", tc.ms, got, tc.want)
		}
	}
}

// --- RecoveryColor ---

func TestRecoveryColor(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{0, "red"},
		{33, "red"},
		{34, "yellow"},
		{66, "yellow"},
		{67, "green"},
		{100, "green"},
	}
	for _, tc := range tests {
		got := RecoveryColor(tc.score)
		if got != tc.want {
			t.Errorf("RecoveryColor(%.0f) = %q, want %q", tc.score, got, tc.want)
		}
	}
}

// --- StrainCategory ---

func TestStrainCategory(t *testing.T) {
	tests := []struct {
		strain float64
		want   string
	}{
		{0, "Minimal"},
		{6.9, "Minimal"},
		{7, "Light"},
		{9.9, "Light"},
		{10, "Moderate"},
		{13.9, "Moderate"},
		{14, "Strenuous"},
		{17.9, "Strenuous"},
		{18, "All Out"},
		{21, "All Out"},
	}
	for _, tc := range tests {
		got := StrainCategory(tc.strain)
		if got != tc.want {
			t.Errorf("StrainCategory(%.1f) = %q, want %q", tc.strain, got, tc.want)
		}
	}
}

// --- SportName ---

func TestSportName(t *testing.T) {
	if got := SportName(0); got != "Running" {
		t.Errorf("SportName(0) = %q, want \"Running\"", got)
	}
	if got := SportName(44); got != "Yoga" {
		t.Errorf("SportName(44) = %q, want \"Yoga\"", got)
	}
	if got := SportName(9999); got != "Sport(9999)" {
		t.Errorf("SportName(9999) = %q, want \"Sport(9999)\"", got)
	}
}

// --- Date navigation helpers ---

func TestPrevNextDay(t *testing.T) {
	ref := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	if got := PrevDay(ref); got != "2026-02-09" {
		t.Errorf("PrevDay = %q, want 2026-02-09", got)
	}
	if got := NextDay(ref); got != "2026-02-11" {
		t.Errorf("NextDay = %q, want 2026-02-11", got)
	}
}

func TestISOWeekStr(t *testing.T) {
	// 2026-02-09 is a Monday in ISO week 7
	ref := time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC)
	if got := ISOWeekStr(ref); got != "2026-W07" {
		t.Errorf("ISOWeekStr(2026-02-09) = %q, want 2026-W07", got)
	}
	if got := PrevWeekStr(ref); got != "2026-W06" {
		t.Errorf("PrevWeekStr(2026-02-09) = %q, want 2026-W06", got)
	}
	if got := NextWeekStr(ref); got != "2026-W08" {
		t.Errorf("NextWeekStr(2026-02-09) = %q, want 2026-W08", got)
	}
}

func TestYearHelpers(t *testing.T) {
	// Jan 1, 2026 is a Thursday; the previous day is in 2025
	ref := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := PrevDayYear(ref); got != 2025 {
		t.Errorf("PrevDayYear(2026-01-01) = %d, want 2025", got)
	}
	if got := NextDayYear(ref); got != 2026 {
		t.Errorf("NextDayYear(2026-01-01) = %d, want 2026", got)
	}

	// Dec 31, 2018 is a Monday; it belongs to ISO week 1 of 2019
	crossYear := time.Date(2018, 12, 31, 0, 0, 0, 0, time.UTC)
	if got := ISOWeekYear(crossYear); got != 2019 {
		t.Errorf("ISOWeekYear(2018-12-31) = %d, want 2019 (cross-year ISO week)", got)
	}
	if got := PrevWeekYear(crossYear); got != 2018 {
		t.Errorf("PrevWeekYear(2018-12-31) = %d, want 2018", got)
	}
	if got := NextWeekYear(crossYear); got != 2019 {
		t.Errorf("NextWeekYear(2018-12-31) = %d, want 2019", got)
	}
}

// --- PrimarySleep ---

func TestPrimarySleep(t *testing.T) {
	t.Run("returns longest non-nap", func(t *testing.T) {
		sleeps := []models.Sleep{
			{Nap: true, Score: models.SleepScore{StageSummary: models.SleepStageSummary{TotalInBedTimeMilli: 9_000_000}}},
			{Nap: false, Score: models.SleepScore{StageSummary: models.SleepStageSummary{TotalInBedTimeMilli: 25_200_000}}},
			{Nap: false, Score: models.SleepScore{StageSummary: models.SleepStageSummary{TotalInBedTimeMilli: 28_800_000}}},
		}
		got := PrimarySleep(sleeps)
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Score.StageSummary.TotalInBedTimeMilli != 28_800_000 {
			t.Errorf("got %d ms, want 28_800_000", got.Score.StageSummary.TotalInBedTimeMilli)
		}
	})

	t.Run("nil when all naps", func(t *testing.T) {
		if got := PrimarySleep([]models.Sleep{{Nap: true}}); got != nil {
			t.Error("expected nil when all naps")
		}
	})

	t.Run("nil on empty", func(t *testing.T) {
		if got := PrimarySleep(nil); got != nil {
			t.Error("expected nil on empty slice")
		}
	})
}

// --- NonNapSleeps ---

func TestNonNapSleeps(t *testing.T) {
	sleeps := []models.Sleep{
		{Nap: false, ID: "a"},
		{Nap: true, ID: "nap1"},
		{Nap: false, ID: "b"},
		{Nap: true, ID: "nap2"},
	}
	got := NonNapSleeps(sleeps)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Index != 0 || got[0].Sleep.ID != "a" {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[1].Index != 1 || got[1].Sleep.ID != "b" {
		t.Errorf("got[1] = %+v", got[1])
	}
}

func TestNonNapSleeps_Empty(t *testing.T) {
	if got := NonNapSleeps(nil); len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

// --- hrvTrendLabel ---

func TestHRVTrendLabel(t *testing.T) {
	t.Run("insufficient data", func(t *testing.T) {
		if got := hrvTrendLabel([]float64{50, 55}); got != "Insufficient data" {
			t.Errorf("got %q, want \"Insufficient data\"", got)
		}
		if got := hrvTrendLabel(nil); got != "Insufficient data" {
			t.Errorf("got %q, want \"Insufficient data\"", got)
		}
	})

	t.Run("stable (flat values)", func(t *testing.T) {
		vals := []float64{50, 50, 50, 50, 50}
		if got := hrvTrendLabel(vals); got != "Stable" {
			t.Errorf("got %q, want Stable", got)
		}
	})

	t.Run("improving (strongly increasing)", func(t *testing.T) {
		vals := []float64{40, 50, 60, 70, 80, 90, 100}
		got := hrvTrendLabel(vals)
		if !strings.HasPrefix(got, "Improving") {
			t.Errorf("got %q, want prefix \"Improving\"", got)
		}
	})

	t.Run("declining (strongly decreasing)", func(t *testing.T) {
		vals := []float64{100, 90, 80, 70, 60, 50, 40}
		got := hrvTrendLabel(vals)
		if !strings.HasPrefix(got, "Declining") {
			t.Errorf("got %q, want prefix \"Declining\"", got)
		}
	})
}

// --- helpers for constructing test data ---

func makeRecovery(score float64) *models.Recovery {
	return &models.Recovery{
		ScoreState: "SCORED",
		Score: models.RecoveryScore{
			RecoveryScore:    score,
			HrvRmssdMilli:    50,
			RestingHeartRate: 55,
		},
	}
}

func makeCycle(strain float64) *models.Cycle {
	return &models.Cycle{
		ScoreState: "SCORED",
		Score:      models.CycleScore{Strain: strain},
	}
}

func makeSleep(ms int64) models.Sleep {
	return models.Sleep{
		ScoreState: "SCORED",
		Nap:        false,
		Score: models.SleepScore{
			StageSummary: models.SleepStageSummary{TotalInBedTimeMilli: ms},
		},
	}
}

// --- BuildWeekStats ---

func TestBuildWeekStats_Empty(t *testing.T) {
	ws := BuildWeekStats(nil)
	if ws.AvgRecovery != 0 || ws.TotalWorkouts != 0 {
		t.Errorf("expected zero stats for empty days: %+v", ws)
	}
}

func TestBuildWeekStats_BasicAggregation(t *testing.T) {
	days := []fetch.DayData{
		{
			Date:     time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC),
			Cycle:    makeCycle(10),
			Recovery: makeRecovery(80), // green
			Sleeps:   []models.Sleep{makeSleep(28_800_000)},
			Workouts: []models.Workout{{}, {}},
		},
		{
			Date:     time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
			Cycle:    makeCycle(12),
			Recovery: makeRecovery(40), // yellow
			Sleeps:   []models.Sleep{makeSleep(25_200_000)},
		},
	}

	ws := BuildWeekStats(days)

	if ws.AvgRecovery != 60 {
		t.Errorf("AvgRecovery = %.1f, want 60.0", ws.AvgRecovery)
	}
	if ws.AvgStrain != 11 {
		t.Errorf("AvgStrain = %.1f, want 11.0", ws.AvgStrain)
	}
	if ws.GreenDays != 1 {
		t.Errorf("GreenDays = %d, want 1", ws.GreenDays)
	}
	if ws.YellowDays != 1 {
		t.Errorf("YellowDays = %d, want 1", ws.YellowDays)
	}
	if ws.RedDays != 0 {
		t.Errorf("RedDays = %d, want 0", ws.RedDays)
	}
	if ws.TotalWorkouts != 2 {
		t.Errorf("TotalWorkouts = %d, want 2", ws.TotalWorkouts)
	}
	if ws.AvgSleepMillis != 27_000_000 {
		t.Errorf("AvgSleepMillis = %d, want 27_000_000 (7h 30m)", ws.AvgSleepMillis)
	}
	if ws.BestDay == nil || ws.BestDay.Recovery.Score.RecoveryScore != 80 {
		t.Error("BestDay should have recovery score 80")
	}
	if ws.WorstDay == nil || ws.WorstDay.Recovery.Score.RecoveryScore != 40 {
		t.Error("WorstDay should have recovery score 40")
	}
}

func TestBuildWeekStats_SkipsUnscored(t *testing.T) {
	days := []fetch.DayData{
		{
			Date: time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC),
			Recovery: &models.Recovery{
				ScoreState: "PENDING_SCORE",
				Score:      models.RecoveryScore{RecoveryScore: 90},
			},
		},
	}
	ws := BuildWeekStats(days)
	if ws.AvgRecovery != 0 {
		t.Errorf("should skip PENDING_SCORE recovery, got AvgRecovery=%.1f", ws.AvgRecovery)
	}
	if ws.GreenDays != 0 {
		t.Errorf("should not count unscored day as green, got GreenDays=%d", ws.GreenDays)
	}
}

func TestBuildWeekStats_NapsExcludedFromSleep(t *testing.T) {
	nap := makeSleep(3_600_000) // 1h
	nap.Nap = true
	main := makeSleep(28_800_000) // 8h

	days := []fetch.DayData{
		{
			Date:   time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC),
			Sleeps: []models.Sleep{main, nap},
		},
	}
	ws := BuildWeekStats(days)
	if ws.AvgSleepMillis != 28_800_000 {
		t.Errorf("AvgSleepMillis = %d, want 28_800_000 (8h), nap should be excluded", ws.AvgSleepMillis)
	}
}

// --- RenderDaily (integration: minimal template) ---

const minimalDailyTmpl = `{{define "daily.md.tmpl"}}date: {{.Date.Format "2006-01-02"}}{{end}}`

func TestRenderDaily(t *testing.T) {
	dir := t.TempDir()
	tmplPath := filepath.Join(dir, "daily.md.tmpl")
	if err := os.WriteFile(tmplPath, []byte(minimalDailyTmpl), 0644); err != nil {
		t.Fatal(err)
	}

	data := fetch.DayData{Date: time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)}
	got, err := RenderDaily(data, tmplPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "2026-02-10") {
		t.Errorf("output missing expected date: %q", got)
	}
}

func TestRenderDaily_MissingTemplate(t *testing.T) {
	_, err := RenderDaily(fetch.DayData{}, "/nonexistent/daily.md.tmpl")
	if err == nil {
		t.Error("expected error for missing template")
	}
}

// --- RenderPersonaSection ---

func TestRenderPersonaSection_EmptyInput(t *testing.T) {
	_, err := RenderPersonaSection(nil)
	if err == nil {
		t.Error("expected error on nil input")
	}
}

func TestRenderPersonaSection_Smoke(t *testing.T) {
	days := []fetch.DayData{
		{
			Date:     time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
			Recovery: makeRecovery(75),
			Sleeps:   []models.Sleep{makeSleep(28_800_000)},
			Cycle:    makeCycle(10),
		},
		{
			Date:     time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC),
			Recovery: makeRecovery(30), // red day
			Sleeps:   []models.Sleep{makeSleep(21_600_000)},
			Cycle:    makeCycle(14),
		},
	}

	got, err := RenderPersonaSection(days)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"WHOOP Health Persona",
		"2026-02-10",
		"2026-02-11",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q", want)
		}
	}
}
