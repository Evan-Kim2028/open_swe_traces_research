package github

import (
	"net/http"
	"testing"
)

// TestDetail01: each [unix_ts, additions, deletions] row becomes a
// *WeeklyStats — element 0 → Week (*Timestamp), 1 → Additions, 2 →
// Deletions.
func TestDetail01(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/stats/code_frequency", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[[1700000000,10,5],[1700001000,3,2]]`))
	})

	weeks, _, err := client.Repositories.ListCodeFrequency(t.Context(), "o", "r")
	if err != nil {
		t.Fatalf("ListCodeFrequency: %v", err)
	}
	if len(weeks) != 2 {
		t.Fatalf("got %d weeks, want 2", len(weeks))
	}
	if weeks[0].Week == nil || weeks[0].Week.Unix() != 1700000000 {
		t.Fatalf("Week = %v, want unix 1700000000", weeks[0].Week)
	}
	if weeks[0].GetAdditions() != 10 || weeks[0].GetDeletions() != 5 {
		t.Fatalf("row = %+v, want additions=10 deletions=5", weeks[0])
	}
	if weeks[1].GetAdditions() != 3 || weeks[1].GetDeletions() != 2 {
		t.Fatalf("row 1 = %+v", weeks[1])
	}
}

// TestDetail02: each [day, hour, commits] row becomes a *PunchCard —
// 0 → Day, 1 → Hour, 2 → Commits.
func TestDetail02(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/stats/punch_card", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[[1,9,4],[3,15,7]]`))
	})

	cards, _, err := client.Repositories.ListPunchCard(t.Context(), "o", "r")
	if err != nil {
		t.Fatalf("ListPunchCard: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("got %d cards, want 2", len(cards))
	}
	if cards[0].GetDay() != 1 || cards[0].GetHour() != 9 || cards[0].GetCommits() != 4 {
		t.Fatalf("row 0 = %+v, want day=1 hour=9 commits=4", cards[0])
	}
	if cards[1].GetDay() != 3 || cards[1].GetHour() != 15 || cards[1].GetCommits() != 7 {
		t.Fatalf("row 1 = %+v", cards[1])
	}
}

// TestDetail03: rows whose length isn't exactly 3 are skipped, not an error.
func TestDetail03(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/stats/code_frequency", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[[1700000000,10,5],[1,2],[1,2,3,4],[]]`))
	})
	mux.HandleFunc("/repos/o/r/stats/punch_card", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[[1,9],[1,9,4]]`))
	})

	weeks, _, err := client.Repositories.ListCodeFrequency(t.Context(), "o", "r")
	if err != nil {
		t.Fatalf("ListCodeFrequency with short rows: %v", err)
	}
	if len(weeks) != 1 || weeks[0].GetAdditions() != 10 {
		t.Fatalf("short/extra rows not skipped: %+v", weeks)
	}

	cards, _, err := client.Repositories.ListPunchCard(t.Context(), "o", "r")
	if err != nil {
		t.Fatalf("ListPunchCard with short rows: %v", err)
	}
	if len(cards) != 1 || cards[0].GetCommits() != 4 {
		t.Fatalf("short rows not skipped: %+v", cards)
	}
}

// TestDetail04: the result is nil/empty when the API returns no rows.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/stats/code_frequency", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	mux.HandleFunc("/repos/o/r/stats/punch_card", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})

	weeks, _, err := client.Repositories.ListCodeFrequency(t.Context(), "o", "r")
	if err != nil {
		t.Fatalf("ListCodeFrequency: %v", err)
	}
	if len(weeks) != 0 {
		t.Fatalf("weeks = %+v, want empty", weeks)
	}

	cards, _, err := client.Repositories.ListPunchCard(t.Context(), "o", "r")
	if err != nil {
		t.Fatalf("ListPunchCard: %v", err)
	}
	if len(cards) != 0 {
		t.Fatalf("cards = %+v, want empty", cards)
	}
}

