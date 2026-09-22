// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-github/v92/github"
)

func decodeTimestamp(t *testing.T, raw string) (github.Timestamp, error) {
	t.Helper()
	var ts github.Timestamp
	err := json.Unmarshal([]byte(raw), &ts)
	return ts, err
}

// Detail 1: quoted RFC3339 strings decode as usual, including
// fractional-second forms. (Inferable: doc)
func TestDetail01(t *testing.T) {
	want := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
	for _, raw := range []string{
		`"2006-01-02T15:04:05Z"`,
		`"2006-01-02T15:04:05.000Z"`,
		`"2006-01-02T16:04:05+01:00"`,
	} {
		ts, err := decodeTimestamp(t, raw)
		if err != nil {
			t.Fatalf("json.Unmarshal(%s) returned error: %v", raw, err)
		}
		if !ts.Time.Equal(want) {
			t.Errorf("json.Unmarshal(%s) = %v, want %v", raw, ts.Time, want)
		}
	}
	ts, err := decodeTimestamp(t, `"2006-01-02T15:04:05.999Z"`)
	if err != nil {
		t.Fatalf("json.Unmarshal fractional form returned error: %v", err)
	}
	if !ts.Time.Equal(want.Add(999 * time.Millisecond)) {
		t.Errorf("fractional-second decode = %v, want %v", ts.Time, want.Add(999*time.Millisecond))
	}
}

// Detail 2: a bare JSON number decodes as a Unix-seconds timestamp.
// (Inferable: doc)
func TestDetail02(t *testing.T) {
	ts, err := decodeTimestamp(t, `1136214245`)
	if err != nil {
		t.Fatalf("json.Unmarshal(1136214245) returned error: %v", err)
	}
	if ts.Unix() != 1136214245 {
		t.Errorf("decode = %v (Unix %d), want Unix 1136214245", ts.Time, ts.Unix())
	}
	if !ts.Time.Equal(time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)) {
		t.Errorf("decode = %v, want 2006-01-02T15:04:05Z", ts.Time)
	}
}

// Detail 3: a bare number too large to be a sane Unix-seconds value decodes
// with millisecond granularity. Inferable: no — asserted as shape only:
// 13-digit values decode without error to the millisecond instant; the
// decoded-year trigger rule (which inputs flip granularity) is NOT pinned.
func TestDetail03(t *testing.T) {
	cases := []struct {
		raw      string
		wantUnix int64
		wantNsec int
	}{
		{`1136214245000`, 1136214245, 0},
		{`1615077308538`, 1615077308, 538000000},
		{`1136214245001`, 1136214245, 1000000},
	}
	for _, c := range cases {
		ts, err := decodeTimestamp(t, c.raw)
		if err != nil {
			t.Fatalf("json.Unmarshal(%s) returned error: %v", c.raw, err)
		}
		if ts.Unix() != c.wantUnix || ts.Nanosecond() != c.wantNsec {
			t.Errorf("json.Unmarshal(%s) = %v (Unix %d, nsec %d), want Unix %d, nsec %d",
				c.raw, ts.Time, ts.Unix(), ts.Nanosecond(), c.wantUnix, c.wantNsec)
		}
	}
}

// Detail 4: a value at the edge of the seconds range still decodes as
// seconds. Inferable: no — asserted as shape only: the boundary input
// 32503680000 must decode without error to a far-future instant
// (year > 2286), not to a near-epoch artifact. The exact boundary location
// and its strictness are NOT pinned.
func TestDetail04(t *testing.T) {
	ts, err := decodeTimestamp(t, `32503680000`)
	if err != nil {
		t.Fatalf("json.Unmarshal(32503680000) returned error: %v", err)
	}
	if ts.Year() <= 2286 {
		t.Errorf("json.Unmarshal(32503680000) decoded to %v (year %d), want a far-future instant", ts.Time, ts.Year())
	}
}

// Detail 5: 11-digit values whose seconds reading lands in years 2286-3000
// decode as seconds, not milliseconds — the granularity switch is on the
// decoded instant's year, not on digit count or magnitude. Inferable: no —
// asserted as shape only: interior window inputs must decode to their
// seconds instant (Unix() == input); the window's exact edges are NOT pinned.
func TestDetail05(t *testing.T) {
	for _, n := range []int64{30000000000, 20000000000, 10000000000} {
		ts, err := decodeTimestamp(t, fmt.Sprint(n))
		if err != nil {
			t.Fatalf("json.Unmarshal(%d) returned error: %v", n, err)
		}
		if ts.Unix() != n {
			t.Errorf("json.Unmarshal(%d) decoded to %v (Unix %d), want the seconds instant Unix %d",
				n, ts.Time, ts.Unix(), n)
		}
	}
}

// Detail 6: 0 decodes to the Unix epoch, not zero-time and not an error.
// (Inferable: partially — the epoch follows from seconds decoding.)
func TestDetail06(t *testing.T) {
	ts, err := decodeTimestamp(t, `0`)
	if err != nil {
		t.Fatalf("json.Unmarshal(0) returned error: %v", err)
	}
	if !ts.Time.Equal(time.Unix(0, 0)) {
		t.Errorf("json.Unmarshal(0) = %v, want Unix epoch %v", ts.Time, time.Unix(0, 0))
	}
}

// Detail 7: quoted non-times, quoted digits, and a literal null are decode
// errors — the error is returned, not swallowed into zero-time.
// (Inferable: partially — erroring is asserted; message content is not pinned.)
func TestDetail07(t *testing.T) {
	for _, raw := range []string{`"asdf"`, `"1234"`, `null`} {
		var ts github.Timestamp
		if err := json.Unmarshal([]byte(raw), &ts); err == nil {
			t.Errorf("json.Unmarshal(%s) succeeded (%v), want a decode error", raw, ts.Time)
		}
	}
}
