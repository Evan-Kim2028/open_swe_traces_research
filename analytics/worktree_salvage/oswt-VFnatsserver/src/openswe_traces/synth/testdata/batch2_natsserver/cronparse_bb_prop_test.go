// Hidden black-box property suite for the cronparse unit.
// Drives only the API in api.md: parseCron, getField, getRange,
// parseIntOrName, mustParseInt, getBits, dayMatches — plus the remaining
// bounds vars and starBit.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package server

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const bbCronHiddenSeed = 20260919

func bbCronSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbCronHiddenSeed
}

func bbCronRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbCronSeed()))
}

func bbCronTs(y int, mo time.Month, d, h, mi, s int) int64 {
	return time.Date(y, mo, d, h, mi, s, 0, time.UTC).UnixNano()
}

// Detail 1: pattern must have EXACTLY 6 whitespace-separated fields
// (sec,min,hour,dom,month,dow); other counts -> error.
func TestDetail01_SixFieldsExactly(t *testing.T) {
	base := bbCronTs(2026, 3, 10, 12, 0, 0)
	for _, p := range []string{
		"* * * * *",
		"* * * * * * *",
		"* * * *",
		"0 0 0 1 1",
		"",
		"*",
	} {
		if _, err := parseCron(p, nil, base); err == nil {
			t.Fatalf("pattern %q should error", p)
		}
	}
	// Exactly 6 parses.
	if _, err := parseCron("* * * * * *", nil, base); err != nil {
		t.Fatalf("6-field pattern: %v", err)
	}
	// Extra whitespace still counts fields, not characters.
	if _, err := parseCron("*  * *  * *  *", nil, base); err != nil {
		t.Fatalf("spaced 6-field: %v", err)
	}
}

// Detail 2: field = comma-separated terms OR'd into a bit set; each term is
// */?/number/range/name + optional /step.
func TestDetail02_FieldOrBits(t *testing.T) {
	got, err := getField("1,2,3", minutes)
	if err != nil {
		t.Fatal(err)
	}
	want := uint64(1<<1 | 1<<2 | 1<<3)
	if got != want {
		t.Fatalf("getField(1,2,3)=%b want %b", got, want)
	}
	// OR'ing overlapping terms.
	got, err = getField("1-3,2-4", minutes)
	if err != nil {
		t.Fatal(err)
	}
	want = uint64(1<<1 | 1<<2 | 1<<3 | 1<<4)
	if got != want {
		t.Fatalf("overlap=%b want %b", got, want)
	}
	// Mixing names, ranges, steps in one field. `jul/1` is the single-bound
	// form: jul through the field max stepping by 1.
	got, err = getField("jan,mar-may,jul/1", months)
	if err != nil {
		t.Fatal(err)
	}
	want = uint64(1<<1 | 1<<3 | 1<<4 | 1<<5 | 1<<7 | 1<<8 | 1<<9 | 1<<10 | 1<<11 | 1<<12)
	if got != want {
		t.Fatalf("names mix=%b want %b", got, want)
	}
	// Star inside a comma list sets the star marker.
	got, err = getField("*,5", minutes)
	if err != nil {
		t.Fatal(err)
	}
	if got&starBit == 0 {
		t.Fatal("star in list should set starBit")
	}
}

// Detail 3: '*' and '?' are equivalent full-range wildcards and set the
// star marker on that field.
func TestDetail03_StarAndQuestion(t *testing.T) {
	star, err := getField("*", seconds)
	if err != nil {
		t.Fatal(err)
	}
	q, err := getField("?", seconds)
	if err != nil {
		t.Fatal(err)
	}
	if star != q {
		t.Fatalf("'*'=%b vs '?'=%b — not equivalent", star, q)
	}
	if star&starBit == 0 {
		t.Fatal("'*' should set starBit")
	}
	// Full range covered.
	full := getBits(seconds.min, seconds.max, 1) | starBit
	if star != full {
		t.Fatalf("'*'=%b want full range|starBit %b", star, full)
	}
	for _, b := range []bounds{minutes, hours, dom, months, dow} {
		got, err := getField("?", b)
		if err != nil {
			t.Fatal(err)
		}
		if got != getBits(b.min, b.max, 1)|starBit {
			t.Fatalf("'?' on bounds %v -> %b", b, got)
		}
	}
}

// Detail 4: bounds — sec/min 0-59, hour 0-23, dom 1-31, month 1-12,
// dow 0-6; names jan-dec and sun-sat, case-insensitive.
func TestDetail04_BoundsAndNames(t *testing.T) {
	if seconds.min != 0 || seconds.max != 59 || minutes.min != 0 || minutes.max != 59 {
		t.Fatal("sec/min bounds")
	}
	if hours.min != 0 || hours.max != 23 || dom.min != 1 || dom.max != 31 {
		t.Fatal("hour/dom bounds")
	}
	if months.min != 1 || months.max != 12 || dow.min != 0 || dow.max != 6 {
		t.Fatal("month/dow bounds")
	}
	monthNames := map[string]uint{"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
		"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12}
	for name, want := range monthNames {
		got, err := parseIntOrName(name, months.names)
		if err != nil || got != want {
			t.Fatalf("month %q=(%d,%v) want %d", name, got, err, want)
		}
		// Case-insensitive.
		got, err = parseIntOrName(strings.ToUpper(name), months.names)
		if err != nil || got != want {
			t.Fatalf("month %q upper=(%d,%v) want %d", name, got, err, want)
		}
	}
	dowNames := map[string]uint{"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6}
	for name, want := range dowNames {
		got, err := parseIntOrName(name, dow.names)
		if err != nil || got != want {
			t.Fatalf("dow %q=(%d,%v) want %d", name, got, err, want)
		}
		got, err = parseIntOrName(strings.Title(name), dow.names)
		if err != nil || got != want {
			t.Fatalf("dow %q title=(%d,%v) want %d", name, got, err, want)
		}
	}
	// Numeric values still work via parseIntOrName.
	got, err := parseIntOrName("7", months.names)
	if err != nil || got != 7 {
		t.Fatalf("numeric=(%d,%v)", got, err)
	}
	// Unknown name -> error.
	if _, err := parseIntOrName("foo", months.names); err == nil {
		t.Fatal("unknown name should error")
	}
	// Names in field position produce bits.
	got64, err := getField("mon,wed,fri", dow)
	if err != nil {
		t.Fatal(err)
	}
	want := uint64(1<<1 | 1<<3 | 1<<5)
	if got64 != want {
		t.Fatalf("dow names=%b want %b", got64, want)
	}
	// Range with names.
	got64, err = getField("mon-fri", dow)
	if err != nil {
		t.Fatal(err)
	}
	want = uint64(1<<1 | 1<<2 | 1<<3 | 1<<4 | 1<<5)
	if got64 != want {
		t.Fatalf("mon-fri=%b want %b", got64, want)
	}
}

// Detail 5: 'n/step' single bound -> n-max/step; step must be >0;
// 'a-b/step' steps inside the range.
func TestDetail05_Steps(t *testing.T) {
	// n/step single bound: n-max.
	got, err := getRange("5/10", minutes)
	if err != nil {
		t.Fatal(err)
	}
	want := getBits(5, 59, 10)
	if got != want {
		t.Fatalf("5/10=%b want %b", got, want)
	}
	// a-b/step.
	got, err = getRange("10-20/3", minutes)
	if err != nil {
		t.Fatal(err)
	}
	want = getBits(10, 20, 3)
	if got != want {
		t.Fatalf("10-20/3=%b want %b", got, want)
	}
	// Step > max-min just takes the start.
	got, err = getRange("5/60", minutes)
	if err != nil {
		t.Fatal(err)
	}
	want = uint64(1 << 5)
	if got != want {
		t.Fatalf("5/60=%b want %b", got, want)
	}
	// step=0 -> error.
	for _, e := range []string{"*/0", "5/0", "1-9/0"} {
		if _, err := getRange(e, minutes); err == nil {
			t.Fatalf("%q step=0 should error", e)
		}
	}
	// getBits itself.
	if got := getBits(0, 5, 2); got != uint64(1|4|16) {
		t.Fatalf("getBits(0,5,2)=%b want bits 0,2,4", got)
	}
	if got := getBits(1, 31, 1); got != 0xFFFFFFFE {
		t.Fatalf("getBits(1,31,1)=%x", got)
	}
}

// Detail 6: '*/k' with k>1 LOSES the star marker (treated as explicit
// range) — flips the dom/dow rule.
func TestDetail06_SteppedStarLosesMarker(t *testing.T) {
	got, err := getRange("*/2", seconds)
	if err != nil {
		t.Fatal(err)
	}
	if got&starBit != 0 {
		t.Fatal("*/2 should NOT set starBit")
	}
	want := getBits(0, 59, 2)
	if got != want {
		t.Fatalf("*/2=%b want %b", got, want)
	}
	got, err = getField("*/5", minutes)
	if err != nil {
		t.Fatal(err)
	}
	if got&starBit != 0 {
		t.Fatal("*/5 field should not carry starBit")
	}
	// Behavior flip: pattern with */2 in dom is a RESTRICTED dom (not
	// starred), so dom OR dow matching suffices when dow also restricted.
	// */2 dom + mon dow on a Monday that is an ODD day: dow matches -> ok.
	// ts = Sunday 2026-03-01; next Monday is 03-02 (even day: dom */2 =
	// odd days only). dow mon -> matches via OR.
	next, err := parseCron("0 0 0 */2 * mon", nil, bbCronTs(2026, 3, 1, 0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if next.Weekday() != time.Monday || next.Day() != 2 {
		t.Fatalf("next=%v want Monday 2026-03-02 (OR rule: dow sufficed)", next)
	}
	// Contrast: same shape with '*' dom (starred) requires BOTH — Monday
	// 03-02 is day 2 (even), dom '*' always true -> also matches. So use a
	// case where the flip matters: restricted dom odd-days + restricted
	// dow Monday, on a Tuesday that IS odd -> dom matches via OR.
	next, err = parseCron("0 0 0 */2 * tue", nil, bbCronTs(2026, 3, 1, 0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	// Tuesday 03-03 is day 3 (odd, in */2) -> matches.
	if next.Day() != 3 || next.Weekday() != time.Tuesday {
		t.Fatalf("next=%v want Tuesday 2026-03-03", next)
	}
}

// Detail 7: validation errors — start<min, end>max, start>end, step=0,
// >1 '-', >1 '/', negative numbers, unparseable value/name.
func TestDetail07_ValidationErrors(t *testing.T) {
	for _, e := range []string{
		"60-61",   // end > max for minutes
		"1-60",    // end > max
		"70",      // value > max
		"10-5",    // start > end
		"1/0",     // step 0
		"1-2-3",   // >1 '-'
		"1/2/3",   // >1 '/'
		"-5",      // negative
		"abc",     // unparseable (no names for minutes)
		"5-",      // missing end
		"/5",      // missing start
		"jan-dec", // names don't exist for minutes
	} {
		if _, err := getField(e, minutes); err == nil {
			t.Fatalf("field %q should error", e)
		}
	}
	// start < min for dom.
	if _, err := getField("0-5", dom); err == nil {
		t.Fatal("dom 0-5 should error (start<min)")
	}
	// mustParseInt errors.
	if _, err := mustParseInt("x"); err == nil {
		t.Fatal("mustParseInt(x) should error")
	}
	if _, err := mustParseInt("-1"); err == nil {
		t.Fatal("mustParseInt(-1) should error")
	}
	if v, err := mustParseInt("42"); err != nil || v != 42 {
		t.Fatalf("mustParseInt(42)=(%d,%v)", v, err)
	}
}

// Detail 8: day rule — if EITHER dom or dow is starred -> BOTH must match;
// if NEITHER starred -> either matching suffices.
func TestDetail08_DayMatchesRule(t *testing.T) {
	allDom := getBits(1, 31, 1) | starBit
	allDow := getBits(0, 6, 1) | starBit
	d15 := uint64(1 << 15)
	mon := uint64(1 << 1)
	// Monday 2026-03-02 (day 2).
	monday := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	// Sunday 2026-03-15 (day 15, weekday sun=0).
	sun15 := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	// Both starred -> AND of two full sets -> always true.
	if !dayMatches(allDom, allDow, monday) {
		t.Fatal("both-starred should match")
	}
	// Either starred -> AND: dom {15} + dow starred -> only the 15th.
	if dayMatches(d15, allDow, monday) {
		t.Fatal("starred-dow AND: day 2 should not match dom{15}")
	}
	if !dayMatches(d15, allDow, sun15) {
		t.Fatal("starred-dow AND: day 15 should match")
	}
	// dom starred + dow {mon}: only Mondays.
	if !dayMatches(allDom, mon, monday) {
		t.Fatal("starred-dom AND: Monday should match")
	}
	if dayMatches(allDom, mon, sun15) {
		t.Fatal("starred-dom AND: Sunday should not match dow{mon}")
	}
	// Neither starred -> OR: dom {15} + dow {mon}.
	if !dayMatches(d15, mon, monday) {
		t.Fatal("OR: Monday should match via dow")
	}
	if !dayMatches(d15, mon, sun15) {
		t.Fatal("OR: the 15th should match via dom")
	}
	wed4 := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC) // Wed day 4
	if dayMatches(d15, mon, wed4) {
		t.Fatal("OR: Wednesday day 4 matches neither -> false")
	}
}

// Detail 9: result is strictly after ts and second-aligned; evaluation
// begins at ts truncated to the second + 1s.
func TestDetail09_StrictlyAfterSecondAligned(t *testing.T) {
	rng := bbCronRng(t)
	for i := 0; i < 30; i++ {
		// Random nanosecond-skewed base.
		base := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC).
			Add(time.Duration(rng.Intn(86400)) * time.Second).
			Add(time.Duration(rng.Intn(1e9)) * time.Nanosecond)
		next, err := parseCron("* * * * * *", nil, base.UnixNano())
		if err != nil {
			t.Fatal(err)
		}
		if !next.After(base) {
			t.Fatalf("next=%v not after base %v", next, base)
		}
		if next.Nanosecond() != 0 {
			t.Fatalf("next=%v not second-aligned", next)
		}
		want := base.Truncate(time.Second).Add(time.Second)
		if !next.Equal(want) {
			t.Fatalf("next=%v want %v (trunc+1s)", next, want)
		}
	}
	// ts exactly on a firing second -> NEXT firing, not ts.
	base := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	next, err := parseCron("* * * * * *", nil, base.UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	if !next.Equal(base.Add(time.Second)) {
		t.Fatalf("exact-fire next=%v want +1s", next)
	}
}

// Detail 10: ts is a NANOSECOND timestamp (Unix(0,ts)); nil loc -> UTC;
// evaluation/reporting in loc.
func TestDetail10_TsNanosAndLoc(t *testing.T) {
	// ts=-1ns -> epoch is the next fire (strictly-after rule).
	next, err := parseCron("0 0 0 1 1 *", nil, -1)
	if err != nil {
		t.Fatal(err)
	}
	if next.Year() != 1970 || next.Month() != 1 || next.Day() != 1 {
		t.Fatalf("epoch next=%v", next)
	}
	// nil loc -> UTC result.
	if next.Location() != time.UTC {
		t.Fatalf("nil loc result in %v want UTC", next.Location())
	}
	// loc honored: a fixed UTC instant lands in different local times.
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no tzdata")
	}
	// 2026-01-15 05:00 UTC = 00:00 EST — midnight NY.
	ts := time.Date(2026, 1, 15, 4, 0, 0, 0, time.UTC).UnixNano()
	nextNY, err := parseCron("0 0 0 * * *", nyc, ts)
	if err != nil {
		t.Fatal(err)
	}
	ny := nextNY.In(nyc)
	if ny.Hour() != 0 || ny.Minute() != 0 || ny.Second() != 0 {
		t.Fatalf("NY next=%v not local midnight", ny)
	}
	// Same pattern in UTC differs.
	nextUTC, err := parseCron("0 0 0 * * *", time.UTC, ts)
	if err != nil {
		t.Fatal(err)
	}
	if nextUTC.Equal(nextNY) {
		t.Fatal("UTC and NY midnights should differ")
	}
}

// Detail 11: walk — month->day->hour->minute->second; inner fields reset
// to minima on wrap; multi-field walks land correctly.
func TestDetail11_CalendarWalk(t *testing.T) {
	// Leap day walk: 2027-01-01 -> 2028-02-29.
	next, err := parseCron("0 0 0 29 2 *", nil, bbCronTs(2027, 1, 1, 0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if next.Year() != 2028 || next.Month() != 2 || next.Day() != 29 {
		t.Fatalf("leap next=%v want 2028-02-29", next)
	}
	// Wrap resets inner fields: hour restricted to 5, from 2026-03-10 23:00
	// -> next day at 05:00:00 (min/sec reset).
	next, err = parseCron("0 0 5 * * *", nil, bbCronTs(2026, 3, 10, 23, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if next.Day() != 11 || next.Hour() != 5 || next.Minute() != 0 || next.Second() != 0 {
		t.Fatalf("hour-wrap next=%v want next-day 05:00:00", next)
	}
	// Minute wrap: "30 *" from :45 -> next hour :30.
	next, err = parseCron("0 30 * * * *", nil, bbCronTs(2026, 3, 10, 12, 45, 0))
	if err != nil {
		t.Fatal(err)
	}
	if next.Hour() != 13 || next.Minute() != 30 || next.Second() != 0 {
		t.Fatalf("minute-wrap next=%v", next)
	}
	// Month walk with restricted dom: 31st-only from Feb -> Mar 31.
	next, err = parseCron("0 0 0 31 * *", nil, bbCronTs(2026, 2, 15, 0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if next.Month() != 3 || next.Day() != 31 {
		t.Fatalf("dom-31 next=%v want 2026-03-31", next)
	}
	// Second-only restriction lands within the same minute.
	next, err = parseCron("17 * * * * *", nil, bbCronTs(2026, 3, 10, 12, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if next.Second() != 17 || next.Minute() != 0 {
		t.Fatalf("sec next=%v", next)
	}
}

// Detail 12: give up after 5 years -> "pattern exceeds maximum range"
// error.
func TestDetail12_FiveYearCap(t *testing.T) {
	// Feb 30 never exists; restricted dom + starred dow -> AND -> never.
	_, err := parseCron("0 0 0 30 2 *", nil, bbCronTs(2026, 1, 1, 0, 0, 0))
	if err == nil {
		t.Fatal("impossible pattern should error")
	}
	if !strings.Contains(err.Error(), "maximum range") {
		t.Fatalf("err=%q want 'maximum range'", err)
	}
	// Feb 29 on a non-leap base still resolves within the window.
	next, err := parseCron("0 0 0 29 2 *", nil, bbCronTs(2026, 1, 1, 0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if next.Year() != 2028 {
		t.Fatalf("feb29 next=%v want 2028", next)
	}
}

// Detail 13: bit positions equal field values (bit 0=dow Sunday,
// bit 1=dom day 1, etc.).
func TestDetail13_BitPositions(t *testing.T) {
	got, err := getField("0", dow)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1<<0 {
		t.Fatalf("dow 0=%b want bit0", got)
	}
	got, err = getField("sun", dow)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1<<0 {
		t.Fatalf("dow sun=%b want bit0", got)
	}
	got, err = getField("1", dom)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1<<1 {
		t.Fatalf("dom 1=%b want bit1", got)
	}
	got, err = getField("12", months)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1<<12 {
		t.Fatalf("month 12=%b want bit12", got)
	}
	got, err = getField("59", seconds)
	if err != nil {
		t.Fatal(err)
	}
	if got != uint64(1)<<59 {
		t.Fatalf("sec 59=%b", got)
	}
}

// Detail 14: dispatcher aliases are outside this closure, but their
// translated patterns flow into parseCron — verify the translated forms.
func TestDetail14_TranslatedPatterns(t *testing.T) {
	base := bbCronTs(2026, 3, 10, 12, 0, 0)
	// "@yearly"/"@annually" -> "0 0 0 1 1 *".
	next, err := parseCron("0 0 0 1 1 *", nil, base)
	if err != nil {
		t.Fatal(err)
	}
	if next.Month() != 1 || next.Day() != 1 || next.Year() != 2027 {
		t.Fatalf("yearly next=%v want 2027-01-01", next)
	}
	// "@monthly" -> "0 0 0 1 * *".
	next, err = parseCron("0 0 0 1 * *", nil, base)
	if err != nil {
		t.Fatal(err)
	}
	if next.Day() != 1 || next.Month() != 4 {
		t.Fatalf("monthly next=%v want 2026-04-01", next)
	}
	// "@weekly" -> "0 0 0 * * 0".
	next, err = parseCron("0 0 0 * * 0", nil, base)
	if err != nil {
		t.Fatal(err)
	}
	if next.Weekday() != time.Sunday {
		t.Fatalf("weekly next=%v want Sunday", next)
	}
	// "@daily"/"@midnight" -> "0 0 0 * * *".
	next, err = parseCron("0 0 0 * * *", nil, base)
	if err != nil {
		t.Fatal(err)
	}
	if next.Hour() != 0 || next.Day() != 11 {
		t.Fatalf("daily next=%v want next midnight", next)
	}
	// "@hourly" -> "0 0 * * * *".
	next, err = parseCron("0 0 * * * *", nil, base)
	if err != nil {
		t.Fatal(err)
	}
	if next.Minute() != 0 || next.Second() != 0 || next.Hour() != 13 {
		t.Fatalf("hourly next=%v want 13:00", next)
	}
}
