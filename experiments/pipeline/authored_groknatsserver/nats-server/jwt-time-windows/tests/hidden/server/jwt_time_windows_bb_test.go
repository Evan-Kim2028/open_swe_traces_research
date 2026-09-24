package server

import (
	"testing"
	"time"

	"github.com/nats-io/jwt/v2"
)

func jtwClaims(times ...jwt.TimeRange) *jwt.UserClaims {
	c := &jwt.UserClaims{}
	c.Times = times
	return c
}

func jtwAt(y int, mo time.Month, d, h, mi, s int) time.Time {
	return time.Date(y, mo, d, h, mi, s, 0, time.UTC)
}

// TestDetail01 (yes): nil claims yield (false, 0).
func TestDetail01(t *testing.T) {
	ok, rem := validateTimesAt(nil, time.Now())
	if ok || rem != 0 {
		t.Fatalf("validateTimesAt(nil) = (%v, %v), want (false, 0)", ok, rem)
	}
}

// TestDetail02 (yes): a non-nil claims with an empty Times slice yields
// (true, 0).
func TestDetail02(t *testing.T) {
	ok, rem := validateTimesAt(&jwt.UserClaims{}, time.Now())
	if !ok || rem != 0 {
		t.Fatalf("empty Times = (%v, %v), want (true, 0)", ok, rem)
	}
}

// TestDetail03 (yes): the location is time.Local unless claims.Locale is
// set, in which case the location is loaded and now converted into it.
func TestDetail03(t *testing.T) {
	// 2024-01-15 08:30 UTC is 2024-01-15 22:30 in Pacific/Kiritimati (UTC+14,
	// no DST), inside a 22:00-23:00 window only under that locale.
	now := jtwAt(2024, time.January, 15, 8, 30, 0)
	c := jtwClaims(jwt.TimeRange{Start: "22:00:00", End: "23:00:00"})
	c.Locale = "Pacific/Kiritimati"
	ok, _ := validateTimesAt(c, now)
	if !ok {
		t.Fatal("range covering now in claims.Locale did not match")
	}
	// The same instant is 08:30 in UTC — outside the window — when the
	// container's local zone has zero offset at that instant.
	if _, off := now.In(time.Local).Zone(); off == 0 {
		ok, _ = validateTimesAt(jtwClaims(jwt.TimeRange{Start: "22:00:00", End: "23:00:00"}), now)
		if ok {
			t.Fatal("local-zone evaluation matched a window only valid in the claims locale")
		}
	}
}

// TestDetail04 (yes): a locale that fails to load yields (false, 0).
func TestDetail04(t *testing.T) {
	c := jtwClaims(jwt.TimeRange{Start: "00:00:00", End: "23:59:59"})
	c.Locale = "Bogus/Zone"
	ok, rem := validateTimesAt(c, time.Now())
	if ok || rem != 0 {
		t.Fatalf("bad locale = (%v, %v), want (false, 0)", ok, rem)
	}
}

// TestDetail05 (shape — Inferable: no): each range's Start and End are parsed
// with the clock layout "15:04:05" in that location.
func TestDetail05(t *testing.T) {
	// Seconds must be honoured: a 5-second window containing now matches,
	// and an instant past its second boundary does not.
	now := jtwAt(2024, time.January, 15, 10, 30, 47)
	c := jtwClaims(jwt.TimeRange{Start: "10:30:45", End: "10:30:50"})
	ok, rem := validateTimesAt(c, now)
	if !ok {
		t.Fatal("sub-minute window containing now did not match")
	}
	if rem != 3*time.Second {
		t.Fatalf("remaining = %v, want 3s", rem)
	}
	ok, _ = validateTimesAt(c, jtwAt(2024, time.January, 15, 10, 30, 51))
	if ok {
		t.Fatal("instant one second past the window matched")
	}
}

// TestDetail06 (yes): start and end are re-seated onto now's calendar date
// in that location, with nanoseconds zeroed.
func TestDetail06(t *testing.T) {
	// now carries sub-second precision; a window starting at now's clock
	// second still counts as already open (start ns are zeroed, not copied
	// from now).
	now := jtwAt(2024, time.January, 15, 12, 0, 0).Add(500 * time.Millisecond)
	c := jtwClaims(jwt.TimeRange{Start: "12:00:00", End: "13:00:00"})
	ok, _ := validateTimesAt(c, now)
	if !ok {
		t.Fatal("window opening at now's clock second did not match a ns-offset now")
	}
	// Re-seat: a bare clock window matches regardless of the calendar date
	// now falls on.
	ok, _ = validateTimesAt(c, jtwAt(2031, time.December, 25, 12, 30, 0))
	if !ok {
		t.Fatal("window did not re-seat onto now's date")
	}
}

// TestDetail07 (shape — Inferable: no): a non-wrapping range matches only
// when start.Before(now) && end.After(now) — endpoints are exclusive.
func TestDetail07(t *testing.T) {
	start, end := jtwAt(2024, 1, 15, 11, 0, 0), jtwAt(2024, 1, 15, 13, 0, 0)
	if ok, _ := validateTimeRangeAt(start, end, jtwAt(2024, 1, 15, 12, 0, 0)); !ok {
		t.Fatal("interior instant did not match")
	}
	if ok, _ := validateTimeRangeAt(start, end, start); ok {
		t.Fatal("instant equal to start matched (endpoint must be exclusive)")
	}
	if ok, _ := validateTimeRangeAt(start, end, end); ok {
		t.Fatal("instant equal to end matched (endpoint must be exclusive)")
	}
	if ok, _ := validateTimeRangeAt(start, end, jtwAt(2024, 1, 15, 13, 0, 1)); ok {
		t.Fatal("instant after end matched")
	}
}

// TestDetail08 (doc): remaining time on a match is end - now.
func TestDetail08(t *testing.T) {
	start, end := jtwAt(2024, 1, 15, 11, 0, 0), jtwAt(2024, 1, 15, 13, 0, 0)
	now := jtwAt(2024, 1, 15, 12, 15, 0)
	ok, rem := validateTimeRangeAt(start, end, now)
	if !ok || rem != 45*time.Minute {
		t.Fatalf("(%v, %v), want (true, 45m)", ok, rem)
	}
}

// TestDetail09 (doc): a range wraps midnight when start.After(end).
func TestDetail09(t *testing.T) {
	// 23:00 -> 01:00 wraps: midnight-adjacent instants are inside.
	start, end := jtwAt(2024, 1, 15, 23, 0, 0), jtwAt(2024, 1, 15, 1, 0, 0)
	if ok, _ := validateTimeRangeAt(start, end, jtwAt(2024, 1, 16, 0, 30, 0)); !ok {
		t.Fatal("wrapping range did not match just after midnight")
	}
	// A non-wrapping range does not match outside itself.
	if ok, _ := validateTimeRangeAt(jtwAt(2024, 1, 15, 1, 0, 0), jtwAt(2024, 1, 15, 23, 0, 0), jtwAt(2024, 1, 15, 23, 30, 0)); ok {
		t.Fatal("non-wrapping 01:00-23:00 matched at 23:30")
	}
	// start == end is not a wrap (start is not after end): nothing matches.
	s := jtwAt(2024, 1, 15, 23, 0, 0)
	if ok, _ := validateTimeRangeAt(s, s, jtwAt(2024, 1, 15, 23, 0, 0)); ok {
		t.Fatal("empty start==end range matched")
	}
}

// TestDetail10 (doc): after midnight in a wrapping range, end.After(now) is
// enough — remaining is end - now.
func TestDetail10(t *testing.T) {
	start, end := jtwAt(2024, 1, 16, 23, 0, 0), jtwAt(2024, 1, 16, 1, 0, 0)
	now := jtwAt(2024, 1, 16, 0, 59, 0)
	ok, rem := validateTimeRangeAt(start, end, now)
	if !ok || rem != time.Minute {
		t.Fatalf("(%v, %v), want (true, 1m)", ok, rem)
	}
}

// TestDetail11 (doc): before midnight in a wrapping range, end is advanced
// one calendar day and the exclusive start<now<end test applies.
func TestDetail11(t *testing.T) {
	start, end := jtwAt(2024, 1, 15, 23, 0, 0), jtwAt(2024, 1, 15, 1, 0, 0)
	now := jtwAt(2024, 1, 15, 23, 30, 0)
	ok, rem := validateTimeRangeAt(start, end, now)
	if !ok || rem != 90*time.Minute {
		t.Fatalf("(%v, %v), want (true, 90m)", ok, rem)
	}
	// Inside a wrap, an instant between end and start (mid-morning) is out.
	if ok, _ := validateTimeRangeAt(start, end, jtwAt(2024, 1, 15, 10, 0, 0)); ok {
		t.Fatal("mid-morning instant matched a 23:00-01:00 wrap")
	}
}

// TestDetail12 (partially): when several ranges match, the returned remaining
// duration is the maximum among them.
func TestDetail12(t *testing.T) {
	now := jtwAt(2024, 1, 15, 10, 0, 0)
	c := jtwClaims(
		jwt.TimeRange{Start: "09:00:00", End: "12:00:00"},
		jwt.TimeRange{Start: "09:00:00", End: "15:00:00"},
		jwt.TimeRange{Start: "20:00:00", End: "21:00:00"},
	)
	ok, rem := validateTimesAt(c, now)
	if !ok || rem != 5*time.Hour {
		t.Fatalf("(%v, %v), want (true, 5h) — the widest match", ok, rem)
	}
}

// TestDetail13 (yes): if no range matches, the result is (false, 0).
func TestDetail13(t *testing.T) {
	now := jtwAt(2024, 1, 15, 10, 0, 0)
	c := jtwClaims(
		jwt.TimeRange{Start: "11:00:00", End: "12:00:00"},
		jwt.TimeRange{Start: "13:00:00", End: "14:00:00"},
	)
	ok, rem := validateTimesAt(c, now)
	if ok || rem != 0 {
		t.Fatalf("(%v, %v), want (false, 0)", ok, rem)
	}
}

// TestDetail14 (yes): a start or end string that fails to parse yields
// (false, 0) for the whole call — even when another range matches.
func TestDetail14(t *testing.T) {
	now := jtwAt(2024, 1, 15, 10, 0, 0)
	c := jtwClaims(
		jwt.TimeRange{Start: "09:00:00", End: "15:00:00"},
		jwt.TimeRange{Start: "bogus", End: "16:00:00"},
	)
	ok, rem := validateTimesAt(c, now)
	if ok || rem != 0 {
		t.Fatalf("(%v, %v), want (false, 0) on a bad range string", ok, rem)
	}
	c = jtwClaims(jwt.TimeRange{Start: "09:00:00", End: "bogus"})
	ok, _ = validateTimesAt(c, now)
	if ok {
		t.Fatal("bad end string did not fail the whole call")
	}
}
