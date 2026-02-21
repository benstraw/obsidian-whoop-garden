package fetch

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/benstraw/whoop-garden/internal/client"
	"github.com/benstraw/whoop-garden/internal/models"
)

const whoopTimeLayout = "2006-01-02T15:04:05.999Z"

// DayData aggregates all WHOOP data for a single calendar day.
type DayData struct {
	Date     time.Time
	Cycle    *models.Cycle
	Recovery *models.Recovery
	Sleeps   []models.Sleep
	Workouts []models.Workout
}

// GetUserProfile fetches the authenticated user's profile.
func GetUserProfile(c *client.Client) (*models.UserProfile, error) {
	body, err := c.Get("/user/profile/basic", nil)
	if err != nil {
		return nil, fmt.Errorf("get user profile: %w", err)
	}
	var profile models.UserProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("parse user profile: %w", err)
	}
	return &profile, nil
}

// GetBodyMeasurements fetches the user's body measurements.
func GetBodyMeasurements(c *client.Client) (*models.BodyMeasurements, error) {
	body, err := c.Get("/user/measurement/body", nil)
	if err != nil {
		return nil, fmt.Errorf("get body measurements: %w", err)
	}
	var m models.BodyMeasurements
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("parse body measurements: %w", err)
	}
	return &m, nil
}

// GetCycles fetches all cycles whose start falls in [start, end) with pagination.
func GetCycles(c *client.Client, start, end time.Time) ([]models.Cycle, error) {
	var all []models.Cycle
	nextToken := ""

	for {
		params := url.Values{}
		params.Set("start", start.UTC().Format(time.RFC3339))
		params.Set("end", end.UTC().Format(time.RFC3339))
		if nextToken != "" {
			params.Set("nextToken", nextToken)
		}

		body, err := c.Get("/cycle", params)
		if err != nil {
			if errors.Is(err, client.ErrNotFound) {
				return all, nil
			}
			return nil, fmt.Errorf("get cycles: %w", err)
		}

		var page models.PaginatedResponse[models.Cycle]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parse cycles: %w", err)
		}

		all = append(all, page.Records...)

		if page.NextToken == "" {
			break
		}
		nextToken = page.NextToken
	}

	return all, nil
}

// GetRecoveries fetches all recovery records whose created_at falls in [start, end).
func GetRecoveries(c *client.Client, start, end time.Time) ([]models.Recovery, error) {
	var all []models.Recovery
	nextToken := ""

	for {
		params := url.Values{}
		params.Set("start", start.UTC().Format(time.RFC3339))
		params.Set("end", end.UTC().Format(time.RFC3339))
		if nextToken != "" {
			params.Set("nextToken", nextToken)
		}

		body, err := c.Get("/recovery", params)
		if err != nil {
			if errors.Is(err, client.ErrNotFound) {
				return all, nil
			}
			return nil, fmt.Errorf("get recoveries: %w", err)
		}

		var page models.PaginatedResponse[models.Recovery]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parse recoveries: %w", err)
		}

		all = append(all, page.Records...)

		if page.NextToken == "" {
			break
		}
		nextToken = page.NextToken
	}

	return all, nil
}

// GetSleeps fetches all sleep records whose start falls in [start, end).
func GetSleeps(c *client.Client, start, end time.Time) ([]models.Sleep, error) {
	var all []models.Sleep
	nextToken := ""

	for {
		params := url.Values{}
		params.Set("start", start.UTC().Format(time.RFC3339))
		params.Set("end", end.UTC().Format(time.RFC3339))
		if nextToken != "" {
			params.Set("nextToken", nextToken)
		}

		body, err := c.Get("/activity/sleep", params)
		if err != nil {
			if errors.Is(err, client.ErrNotFound) {
				return all, nil
			}
			return nil, fmt.Errorf("get sleeps: %w", err)
		}

		var page models.PaginatedResponse[models.Sleep]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parse sleeps: %w", err)
		}

		all = append(all, page.Records...)

		if page.NextToken == "" {
			break
		}
		nextToken = page.NextToken
	}

	return all, nil
}

// GetWorkouts fetches all workout records whose start falls in [start, end).
func GetWorkouts(c *client.Client, start, end time.Time) ([]models.Workout, error) {
	var all []models.Workout
	nextToken := ""

	for {
		params := url.Values{}
		params.Set("start", start.UTC().Format(time.RFC3339))
		params.Set("end", end.UTC().Format(time.RFC3339))
		if nextToken != "" {
			params.Set("nextToken", nextToken)
		}

		body, err := c.Get("/activity/workout", params)
		if err != nil {
			if errors.Is(err, client.ErrNotFound) {
				return all, nil
			}
			return nil, fmt.Errorf("get workouts: %w", err)
		}

		var page models.PaginatedResponse[models.Workout]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parse workouts: %w", err)
		}

		all = append(all, page.Records...)

		if page.NextToken == "" {
			break
		}
		nextToken = page.NextToken
	}

	return all, nil
}

// GetDayData fetches and aggregates all WHOOP data for a given calendar date.
//
// WHOOP cycles do not align with calendar-day boundaries — a cycle starts
// when the user wakes up from their overnight sleep. We therefore:
//  1. Query cycles whose start falls in [day 00:00 UTC, day+1 00:00 UTC).
//  2. Concurrently fetch recoveries, sleeps, and workouts bounded to the cycle's
//     time range. Recovery is matched to the cycle via cycle_id.
//  3. Sleep window extends 24h before cycleStart to capture the preceding night.
func GetDayData(c *client.Client, date time.Time) (DayData, error) {
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	nextDay := day.AddDate(0, 0, 1)

	data := DayData{Date: day}

	cycles, err := GetCycles(c, day, nextDay)
	if err != nil {
		return data, err
	}
	if len(cycles) == 0 {
		return data, nil
	}

	// Use the first (most recent) cycle for the day.
	cycle := cycles[0]
	data.Cycle = &cycle

	cycleStart, err := ParseWhoopTime(cycle.Start)
	if err != nil {
		return data, fmt.Errorf("parse cycle start: %w", err)
	}
	cycleEnd := nextDay // default if cycle hasn't ended yet
	if cycle.End != "" {
		if t, err := ParseWhoopTime(cycle.End); err == nil {
			cycleEnd = t
		}
	}

	// Phase 2: fetch recovery, sleeps, and workouts concurrently.
	type recoveriesResult struct {
		v   []models.Recovery
		err error
	}
	type sleepResult struct {
		v   []models.Sleep
		err error
	}
	type workoutResult struct {
		v   []models.Workout
		err error
	}

	recCh := make(chan recoveriesResult, 1)
	sleepCh := make(chan sleepResult, 1)
	workCh := make(chan workoutResult, 1)

	go func() {
		v, err := GetRecoveries(c, cycleStart, cycleEnd)
		recCh <- recoveriesResult{v, err}
	}()

	go func() {
		// Sleep window: 24h before cycle start (captures preceding night's sleep)
		// through cycle end (captures naps during the day).
		sleepStart := cycleStart.Add(-24 * time.Hour)
		v, err := GetSleeps(c, sleepStart, cycleEnd)
		sleepCh <- sleepResult{v, err}
	}()

	go func() {
		v, err := GetWorkouts(c, cycleStart, cycleEnd)
		workCh <- workoutResult{v, err}
	}()

	rr := <-recCh
	if rr.err != nil {
		return data, rr.err
	}
	sr := <-sleepCh
	if sr.err != nil {
		return data, sr.err
	}
	wr := <-workCh
	if wr.err != nil {
		return data, wr.err
	}

	// Pick the recovery whose cycle_id matches this cycle.
	for i := range rr.v {
		if rr.v[i].CycleID == cycle.ID {
			data.Recovery = &rr.v[i]
			break
		}
	}
	data.Sleeps = sr.v
	data.Workouts = wr.v

	return data, nil
}

// ParseWhoopTime parses a WHOOP timestamp string into time.Time.
func ParseWhoopTime(s string) (time.Time, error) {
	t, err := time.Parse(whoopTimeLayout, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	return t, err
}
