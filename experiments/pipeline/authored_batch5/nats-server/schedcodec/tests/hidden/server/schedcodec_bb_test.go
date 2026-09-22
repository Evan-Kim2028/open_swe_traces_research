package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"
)

// TestDetail01: encode wire format — version byte 1, LE u64 count, LE u64
// highSeq; per entry LE u16 subjLen, subject, signed-varint ts, uvarint seq.
func TestDetail01(t *testing.T) {
	ms := newMsgScheduling(func() {})
	ms.init(7, "foo", 12345)
	ms.init(9, "bar", -50)
	b := ms.encode(99)
	le := binary.LittleEndian
	if len(b) < headerLen || b[0] != 1 {
		t.Fatalf("missing version byte: %x", b)
	}
	if le.Uint64(b[1:]) != 2 {
		t.Fatalf("count = %d", le.Uint64(b[1:]))
	}
	if le.Uint64(b[9:]) != 99 {
		t.Fatalf("highSeq stamp = %d", le.Uint64(b[9:]))
	}
	// Walk the entry region: per entry u16 subjLen, subject, varint ts,
	// uvarint seq. Order within the map is not committed; collect entries.
	type ent struct {
		subj string
		ts   int64
		seq  uint64
	}
	var ents []ent
	i := headerLen
	for i < len(b) {
		if i+2 > len(b) {
			t.Fatalf("truncated entry header at %d", i)
		}
		sl := int(le.Uint16(b[i:]))
		i += 2
		if i+sl > len(b) {
			t.Fatalf("truncated subject at %d", i)
		}
		subj := string(b[i : i+sl])
		i += sl
		ts, tn := binary.Varint(b[i:])
		if tn <= 0 {
			t.Fatalf("ts not a signed varint at %d", i)
		}
		i += tn
		seq, un := binary.Uvarint(b[i:])
		if un <= 0 {
			t.Fatalf("seq not a uvarint at %d", i)
		}
		i += un
		ents = append(ents, ent{subj, ts, seq})
	}
	want := map[string]ent{"foo": {"foo", 12345, 7}, "bar": {"bar", -50, 9}}
	if len(ents) != 2 {
		t.Fatalf("decoded %d entries", len(ents))
	}
	for _, e := range ents {
		if w := want[e.subj]; e != w {
			t.Fatalf("entry %+v != %+v", e, w)
		}
	}
}

// TestDetail02: decode errors — short buffer → io.ErrShortBuffer; bad
// version → ErrMsgScheduleInvalidVersion; any truncation past the header
// → io.ErrUnexpectedEOF. Valid buffers populate all bookkeeping and
// return the stamp.
func TestDetail02(t *testing.T) {
	ms := newMsgScheduling(func() {})
	ms.init(7, "foo", 12345)
	ms.init(9, "bar", -50)
	good := ms.encode(99)

	if _, err := ms.decode(good[:headerLen-1]); !errors.Is(err, io.ErrShortBuffer) {
		t.Fatalf("short header: %v", err)
	}
	badv := append([]byte{}, good...)
	badv[0] = 9
	if _, err := ms.decode(badv); !errors.Is(err, ErrMsgScheduleInvalidVersion) {
		t.Fatalf("bad version: %v", err)
	}
	for _, cut := range []int{headerLen, headerLen + 1, headerLen + 4, len(good) - 1} {
		if _, err := newMsgScheduling(func() {}).decode(good[:cut]); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("cut at %d: %v", cut, err)
		}
	}
	ms2 := newMsgScheduling(func() {})
	stamp, err := ms2.decode(good)
	if err != nil || stamp != 99 {
		t.Fatalf("decode stamp = %d err=%v", stamp, err)
	}
	if ms2.schedules["foo"].seq != 7 || ms2.seqToSubj[7] != "foo" ||
		ms2.schedules["bar"].seq != 9 || ms2.seqToSubj[9] != "bar" {
		t.Fatalf("bookkeeping not populated: %v %v", ms2.schedules, ms2.seqToSubj)
	}
}

// TestDetail03: empty pattern is a valid no-op.
func TestDetail03(t *testing.T) {
	ts, repeat, ok := parseMsgSchedule("", nil, 0)
	if !ok || repeat || !ts.IsZero() {
		t.Fatalf("empty pattern = %v %v %v", ts, repeat, ok)
	}
}

// TestDetail04: @at <RFC3339> → one-shot at that instant; @every <dur> →
// repeating with dur ≥ 1s; neither accepts a non-nil location.
func TestDetail04(t *testing.T) {
	at := time.Now().UTC().Add(time.Hour).Round(time.Second)
	ts, repeat, ok := parseMsgSchedule(fmt.Sprintf("@at %s", at.Format(time.RFC3339)), nil, 0)
	if !ok || repeat || !ts.Equal(at) {
		t.Fatalf("@at = %v %v %v, want %v", ts, repeat, ok, at)
	}
	ts, repeat, ok = parseMsgSchedule("@every 5s", nil, at.UnixNano())
	if !ok || !repeat {
		t.Fatalf("@every = %v %v %v", ts, repeat, ok)
	}
	if _, _, ok := parseMsgSchedule("@every 999ms", nil, 0); ok {
		t.Fatal("sub-second interval must be invalid")
	}
	if _, _, ok := parseMsgSchedule("@every bogus", nil, 0); ok {
		t.Fatal("unparsable interval must be invalid")
	}
	loc := time.UTC
	if _, _, ok := parseMsgSchedule(fmt.Sprintf("@at %s", at.Format(time.RFC3339)), loc, 0); ok {
		t.Fatal("@at must reject a location")
	}
	if _, _, ok := parseMsgSchedule("@every 5s", loc, 0); ok {
		t.Fatal("@every must reject a location")
	}
}

// TestDetail05: predefined aliases expand to their cron equivalents; a
// bad spec is invalid.
func TestDetail05(t *testing.T) {
	// Assert via observable next-fire: at a Sunday 00:00:00 base each alias
	// lands on the documented instant.
	base := time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC) // a Sunday
	for base.Weekday() != time.Sunday {
		base = base.AddDate(0, 0, 1)
	}
	cases := map[string]time.Time{
		"@yearly":   time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC),
		"@annually": time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC),
		"@monthly":  time.Date(2030, 7, 1, 0, 0, 0, 0, time.UTC),
		"@weekly":   base.AddDate(0, 0, 7),
		"@daily":    base.AddDate(0, 0, 1),
		"@midnight": base.AddDate(0, 0, 1),
		"@hourly":   base.Add(time.Hour),
	}
	for p, want := range cases {
		ts, repeat, ok := parseMsgSchedule(p, nil, base.UnixNano())
		if !ok || !repeat || !ts.Equal(want) {
			t.Fatalf("%s = %v %v %v, want %v", p, ts, repeat, ok, want)
		}
	}
	if _, _, ok := parseMsgSchedule("not a schedule", nil, base.UnixNano()); ok {
		t.Fatal("bad spec must be invalid")
	}
}

// TestDetail06: a computed next in the past is skipped forward — @every
// re-arms from now rounded to the second plus one interval and stays
// repeating.
func TestDetail06(t *testing.T) {
	now := time.Now().UTC().Round(time.Second)
	ts, repeat, ok := parseMsgSchedule("@every 5s", nil, 0)
	if !ok || !repeat {
		t.Fatalf("@every from epoch: %v %v %v", ts, repeat, ok)
	}
	if ts.Before(now.Add(5*time.Second)) || ts.After(now.Add(6*time.Second)) {
		t.Fatalf("bumped next = %v, want ~%v", ts, now.Add(5*time.Second))
	}
	// Cron path: still repeating, next not in the past.
	ts, repeat, ok = parseMsgSchedule("* * * * * *", nil, 0)
	if !ok || !repeat || ts.Before(now) {
		t.Fatalf("cron from epoch: %v %v %v", ts, repeat, ok)
	}
}

// TestDetail07: @every anchors on the supplied timestamp rounded to the
// second plus the interval — not on wall clock.
func TestDetail07(t *testing.T) {
	anchor := time.Now().UTC().Add(time.Hour).Round(time.Second)
	ts, repeat, ok := parseMsgSchedule("@every 1500ms", nil, anchor.UnixNano())
	if !ok || !repeat {
		t.Fatalf("@every anchored: %v %v %v", ts, repeat, ok)
	}
	want := anchor.Round(time.Second).Add(1500 * time.Millisecond)
	if !ts.Equal(want) {
		t.Fatalf("anchored next = %v, want %v", ts, want)
	}
	// Re-arming from that fire time rounds again before adding.
	ts2, _, _ := parseMsgSchedule("@every 1500ms", nil, ts.UnixNano())
	if !ts2.Equal(ts.Round(time.Second).Add(1500 * time.Millisecond)) {
		t.Fatalf("re-arm = %v", ts2)
	}
}
