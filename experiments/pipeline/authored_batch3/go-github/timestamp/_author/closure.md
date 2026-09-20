# Closure — timestamp

Package: github (root). File: github/timestamp.go.
Removed bodies: Timestamp.UnmarshalJSON — stubbed to a magnitude-threshold
decoder (ParseInt; i > 9999999999 → Unix(0, i*1e6) else Unix(i, 0); RFC3339
Parse fallback). Keeps every in-tree fixture green — rate-limit unix-second
resets, repos-stats week numbers, and the 13-digit millisecond audit-log
created_at values all decode correctly — while replacing gold's real rule
(seconds-decoded year strictly after 3000 → milliseconds). The hidden
divergence is the 11-digit window whose seconds interpretation lands in years
2286–3000 (stub: ms → 1970s; gold: seconds → far future) plus the strict
year-3000 boundary.
Kept: Timestamp.Equal, Timestamp.String, Timestamp.GetTime, marshal side.
Tests removed: 4 funcs in github/timestamp_test.go (TestTimestamp_Unmarshal,
TestWrappedTimestamp_Unmarshal, TestTimestamp_MarshalReflexivity,
TestWrappedTimestamp_MarshalReflexivity). strconv stays used.
