package server

import (
	"testing"
	"time"
)

// TestDetail01 (shape — Inferable: no): an empty pattern is legal and means
// no schedule — zero time, repeating=false, ok=true.
func TestDetail01(t *testing.T) {
	next, rep, ok := parseMsgSchedule("", nil, time.Now().UnixNano())
	if !ok || rep || !next.IsZero() {
		t.Fatalf("empty pattern = (%v, %v, %v), want (zero, false, true)", next, rep, ok)
	}
}

// TestDetail02 (doc): "@at " is a one-shot RFC3339 timestamp from the
// remainder.
func TestDetail02(t *testing.T) {
	want := time.Date(2030, time.June, 15, 10, 0, 0, 0, time.UTC)
	next, rep, ok := parseMsgSchedule("@at 2030-06-15T10:00:00Z", nil, time.Now().UnixNano())
	if !ok || rep || !next.Equal(want) {
		t.Fatalf("@at = (%v, %v, %v), want (%v, false, true)", next, rep, ok, want)
	}
}

// TestDetail03 (doc): @at is illegal when loc is non-nil.
func TestDetail03(t *testing.T) {
	loc, _ := time.LoadLocation("Pacific/Kiritimati")
	_, _, ok := parseMsgSchedule("@at 2030-06-15T10:00:00Z", loc, time.Now().UnixNano())
	if ok {
		t.Fatal("@at with a location was accepted")
	}
}

// TestDetail04 (yes): @at parse failure -> (zero, false, false).
func TestDetail04(t *testing.T) {
	for _, p := range []string{"@at garbage", "@at 2030-13-40T99:99:99Z", "@at "} {
		next, rep, ok := parseMsgSchedule(p, nil, time.Now().UnixNano())
		if ok || rep || !next.IsZero() {
			t.Fatalf("%q = (%v, %v, %v), want (zero, false, false)", p, next, rep, ok)
		}
	}
}

// TestDetail05 (doc): "@every " is a repeating ParseDuration interval from
// the remainder.
func TestDetail05(t *testing.T) {
	ts := time.Now().Add(time.Hour).UnixNano()
	next, rep, ok := parseMsgSchedule("@every 1m", nil, ts)
	want := time.Unix(0, ts).UTC().Round(time.Second).Add(time.Minute)
	if !ok || !rep || !next.Equal(want) {
		t.Fatalf("@every 1m = (%v, %v, %v), want (%v, true, true)", next, rep, ok, want)
	}
}

// TestDetail06 (doc): @every is illegal when loc is non-nil.
func TestDetail06(t *testing.T) {
	loc, _ := time.LoadLocation("Pacific/Kiritimati")
	_, _, ok := parseMsgSchedule("@every 1m", loc, time.Now().UnixNano())
	if ok {
		t.Fatal("@every with a location was accepted")
	}
}

// TestDetail07 (shape — Inferable: no): @every intervals shorter than one
// second are illegal; one second is the floor.
func TestDetail07(t *testing.T) {
	for _, p := range []string{"@every 999ms", "@every 500ms", "@every 1ns"} {
		if _, _, ok := parseMsgSchedule(p, nil, time.Now().UnixNano()); ok {
			t.Fatalf("%q accepted a sub-second interval", p)
		}
	}
	if _, _, ok := parseMsgSchedule("@every 1s", nil, time.Now().UnixNano()); !ok {
		t.Fatal("@every 1s was rejected — one second is the floor")
	}
}

// TestDetail08 (doc): @every computes Unix(0,ts).UTC().Round(Second).Add(dur);
// if that instant is in the past it becomes Now().UTC().Round(Second).Add(dur).
func TestDetail08(t *testing.T) {
	// Future ts: deterministic.
	ts := time.Now().Add(2 * time.Hour).UnixNano()
	next, _, ok := parseMsgSchedule("@every 30s", nil, ts)
	if want := time.Unix(0, ts).UTC().Round(time.Second).Add(30 * time.Second); !ok || !next.Equal(want) {
		t.Fatalf("future ts: got %v, want %v", next, want)
	}
	// Past ts: recomputed from now.
	before := time.Now()
	next, _, ok = parseMsgSchedule("@every 30s", nil, time.Now().Add(-time.Hour).UnixNano())
	after := time.Now()
	if !ok {
		t.Fatal("past ts: not ok")
	}
	lo := before.UTC().Round(time.Second).Add(30 * time.Second)
	hi := after.UTC().Round(time.Second).Add(30 * time.Second)
	if next.Before(lo) || next.After(hi) {
		t.Fatalf("past ts: next=%v outside [%v, %v]", next, lo, hi)
	}
}

// TestDetail09 (shape — Inferable: no): @yearly / @annually are repeating
// schedules firing at the first second of 1 January.
func TestDetail09(t *testing.T) {
	now := time.Now().UTC().Round(time.Second)
	for _, p := range []string{"@yearly", "@annually"} {
		next, rep, ok := parseMsgSchedule(p, nil, now.UnixNano())
		if !ok || !rep {
			t.Fatalf("%s = rep=%v ok=%v", p, rep, ok)
		}
		if next.Month() != time.January || next.Day() != 1 ||
			next.Hour() != 0 || next.Minute() != 0 || next.Second() != 0 || next.Nanosecond() != 0 {
			t.Fatalf("%s fired at %v, want first second of Jan 1", p, next)
		}
		if !next.After(now) || next.After(now.AddDate(1, 0, 0)) {
			t.Fatalf("%s next=%v not within the coming year", p, next)
		}
	}
}

// TestDetail10 (shape — Inferable: no): @monthly is repeating, firing at the
// first second of the first day of the month.
func TestDetail10(t *testing.T) {
	now := time.Now().UTC().Round(time.Second)
	next, rep, ok := parseMsgSchedule("@monthly", nil, now.UnixNano())
	if !ok || !rep {
		t.Fatalf("@monthly = rep=%v ok=%v", rep, ok)
	}
	if next.Day() != 1 || next.Hour() != 0 || next.Minute() != 0 || next.Second() != 0 {
		t.Fatalf("@monthly fired at %v, want first second of the 1st", next)
	}
	if !next.After(now) || next.After(now.AddDate(0, 1, 1)) {
		t.Fatalf("@monthly next=%v not within the coming month", next)
	}
}

// TestDetail11 (shape — Inferable: no): @weekly is repeating, firing at the
// first second of Sunday.
func TestDetail11(t *testing.T) {
	now := time.Now().UTC().Round(time.Second)
	next, rep, ok := parseMsgSchedule("@weekly", nil, now.UnixNano())
	if !ok || !rep {
		t.Fatalf("@weekly = rep=%v ok=%v", rep, ok)
	}
	if next.Weekday() != time.Sunday || next.Hour() != 0 || next.Minute() != 0 || next.Second() != 0 {
		t.Fatalf("@weekly fired at %v, want first second of a Sunday", next)
	}
	if !next.After(now) || next.After(now.AddDate(0, 0, 8)) {
		t.Fatalf("@weekly next=%v not within the coming week", next)
	}
}

// TestDetail12 (shape — Inferable: no): @daily / @midnight are repeating,
// firing at the first second of each day.
func TestDetail12(t *testing.T) {
	now := time.Now().UTC().Round(time.Second)
	for _, p := range []string{"@daily", "@midnight"} {
		next, rep, ok := parseMsgSchedule(p, nil, now.UnixNano())
		if !ok || !rep {
			t.Fatalf("%s = rep=%v ok=%v", p, rep, ok)
		}
		if next.Hour() != 0 || next.Minute() != 0 || next.Second() != 0 {
			t.Fatalf("%s fired at %v, want midnight", p, next)
		}
		if !next.After(now) || next.After(now.Add(25*time.Hour)) {
			t.Fatalf("%s next=%v not within the coming day", p, next)
		}
	}
}

// TestDetail13 (shape — Inferable: no): @hourly is repeating, firing at the
// first second of each hour.
func TestDetail13(t *testing.T) {
	now := time.Now().UTC().Round(time.Second)
	next, rep, ok := parseMsgSchedule("@hourly", nil, now.UnixNano())
	if !ok || !rep {
		t.Fatalf("@hourly = rep=%v ok=%v", rep, ok)
	}
	if next.Minute() != 0 || next.Second() != 0 || next.Nanosecond() != 0 {
		t.Fatalf("@hourly fired at %v, want the top of the hour", next)
	}
	if !next.After(now) || next.After(now.Add(61*time.Minute)) {
		t.Fatalf("@hourly next=%v not within the coming hour", next)
	}
}

// TestDetail14 (yes): any other pattern is handed to the cron parser; a parse
// error is illegal.
func TestDetail14(t *testing.T) {
	// A six-field cron spec dispatches fine.
	next, rep, ok := parseMsgSchedule("0 0 12 * * *", nil, time.Now().UnixNano())
	if !ok || !rep || next.Hour() != 12 || next.Minute() != 0 || next.Second() != 0 {
		t.Fatalf("cron pattern = (%v, %v, %v)", next, rep, ok)
	}
	// Five fields and junk both fail.
	for _, p := range []string{"0 12 * * *", "garbage", "@bogus 1h"} {
		if _, _, ok := parseMsgSchedule(p, nil, time.Now().UnixNano()); ok {
			t.Fatalf("%q was accepted", p)
		}
	}
}

// TestDetail15 (doc): a cron next-time already in the past is recomputed from
// now (UTC, second-rounded).
func TestDetail15(t *testing.T) {
	before := time.Now()
	next, rep, ok := parseMsgSchedule("* * * * * *", nil, time.Now().Add(-time.Hour).UnixNano())
	after := time.Now()
	if !ok || !rep {
		t.Fatalf("every-second cron = rep=%v ok=%v", rep, ok)
	}
	if next.Before(before.UTC().Round(time.Second)) || next.After(after.UTC().Round(time.Second).Add(2*time.Second)) {
		t.Fatalf("next=%v not recomputed from now (window %v..%v)", next, before, after)
	}
}

// TestDetail16 (yes): successful @every and cron results have
// repeating=true.
func TestDetail16(t *testing.T) {
	if _, rep, _ := parseMsgSchedule("@every 2m", nil, time.Now().UnixNano()); !rep {
		t.Fatal("@every did not report repeating")
	}
	if _, rep, _ := parseMsgSchedule("0 0 12 * * *", nil, time.Now().UnixNano()); !rep {
		t.Fatal("cron pattern did not report repeating")
	}
}
