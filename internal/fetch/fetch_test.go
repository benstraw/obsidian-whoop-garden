package fetch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/benstraw/whoop-garden/internal/client"
	"github.com/benstraw/whoop-garden/internal/models"
)

// --- ParseWhoopTime ---

func TestParseWhoopTime(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		wantUTC string
	}{
		// WHOOP-native format (millisecond precision, Z suffix)
		{"2026-02-10T07:30:00.000Z", false, "2026-02-10 07:30:00 +0000 UTC"},
		// RFC3339 (sometimes returned by the API)
		{"2026-02-10T07:30:00Z", false, "2026-02-10 07:30:00 +0000 UTC"},
		// Invalid
		{"not-a-date", true, ""},
		{"2026-13-01T00:00:00Z", true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseWhoopTime(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.UTC().String() != tc.wantUTC {
				t.Errorf("got %q, want %q", got.UTC().String(), tc.wantUTC)
			}
		})
	}
}

// --- GetCycles (paginated) ---

func TestGetCycles_Paginated(t *testing.T) {
	page1 := models.PaginatedResponse[models.Cycle]{
		Records:   []models.Cycle{{ID: 1, ScoreState: "SCORED"}},
		NextToken: "token2",
	}
	page2 := models.PaginatedResponse[models.Cycle]{
		Records:   []models.Cycle{{ID: 2, ScoreState: "SCORED"}},
		NextToken: "",
	}

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("nextToken") == "" {
			json.NewEncoder(w).Encode(page1)
		} else {
			json.NewEncoder(w).Encode(page2)
		}
	}))
	defer srv.Close()

	c := client.NewClientWithBaseURL("tok", srv.URL)
	start := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)

	cycles, err := GetCycles(c, start, end)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 2 {
		t.Errorf("got %d cycles, want 2", len(cycles))
	}
	if callCount != 2 {
		t.Errorf("made %d API calls, want 2 (one per page)", callCount)
	}
	if cycles[0].ID != 1 || cycles[1].ID != 2 {
		t.Errorf("cycle IDs = %d,%d, want 1,2", cycles[0].ID, cycles[1].ID)
	}
}

func TestGetCycles_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := client.NewClientWithBaseURL("tok", srv.URL)
	start := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	cycles, err := GetCycles(c, start, start.AddDate(0, 0, 1))
	if err != nil {
		t.Errorf("404 from cycle endpoint should return empty slice, got error: %v", err)
	}
	if len(cycles) != 0 {
		t.Errorf("expected empty slice on 404, got %d cycles", len(cycles))
	}
}

func TestGetCycles_EmptyPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(models.PaginatedResponse[models.Cycle]{
			Records:   nil,
			NextToken: "",
		})
	}))
	defer srv.Close()

	c := client.NewClientWithBaseURL("tok", srv.URL)
	cycles, err := GetCycles(c, time.Now(), time.Now().AddDate(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 0 {
		t.Errorf("expected 0 cycles for empty page, got %d", len(cycles))
	}
}

// TestGetCycles_QueryParams verifies start/end are forwarded to the API.
func TestGetCycles_QueryParams(t *testing.T) {
	var receivedStart, receivedEnd string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedStart = r.URL.Query().Get("start")
		receivedEnd = r.URL.Query().Get("end")
		json.NewEncoder(w).Encode(models.PaginatedResponse[models.Cycle]{})
	}))
	defer srv.Close()

	c := client.NewClientWithBaseURL("tok", srv.URL)
	start := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)

	if _, err := GetCycles(c, start, end); err != nil {
		t.Fatal(err)
	}
	if receivedStart == "" || receivedEnd == "" {
		t.Errorf("expected start/end query params, got start=%q end=%q", receivedStart, receivedEnd)
	}
}

// --- GetSleeps / GetWorkouts / GetRecoveries ---
// These share the same pagination logic as GetCycles; a smoke test each is sufficient.

func TestGetSleeps_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := client.NewClientWithBaseURL("tok", srv.URL)
	sleeps, err := GetSleeps(c, time.Now(), time.Now().AddDate(0, 0, 1))
	if err != nil {
		t.Errorf("404 should return empty, got error: %v", err)
	}
	if len(sleeps) != 0 {
		t.Errorf("expected 0 sleeps, got %d", len(sleeps))
	}
}

func TestGetWorkouts_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := client.NewClientWithBaseURL("tok", srv.URL)
	workouts, err := GetWorkouts(c, time.Now(), time.Now().AddDate(0, 0, 1))
	if err != nil {
		t.Errorf("404 should return empty, got error: %v", err)
	}
	if len(workouts) != 0 {
		t.Errorf("expected 0 workouts, got %d", len(workouts))
	}
}

func TestGetRecoveries_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := client.NewClientWithBaseURL("tok", srv.URL)
	recoveries, err := GetRecoveries(c, time.Now(), time.Now().AddDate(0, 0, 1))
	if err != nil {
		t.Errorf("404 should return empty, got error: %v", err)
	}
	if len(recoveries) != 0 {
		t.Errorf("expected 0 recoveries, got %d", len(recoveries))
	}
}
