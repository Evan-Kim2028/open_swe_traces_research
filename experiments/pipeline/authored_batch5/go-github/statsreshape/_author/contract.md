# Contract — statsreshape

`RepositoriesService.ListCodeFrequency` and `ListPunchCard` reshape the
API's raw three-element integer rows into typed structures. Every
commitment below is covered by a hidden test; every hidden test maps to a
commitment.

## Commitments

1. **Code frequency reshape.** Each `[unix_ts, additions, deletions]` row
   becomes a `*WeeklyStats` — element 0 → `Week` (a `*Timestamp` of that
   unix time), 1 → `Additions`, 2 → `Deletions`. Covered by
   `TestDetail01`.
2. **Punch card reshape.** Each `[day, hour, commits]` row becomes a
   `*PunchCard` — 0 → `Day`, 1 → `Hour`, 2 → `Commits`. Covered by
   `TestDetail02`.
3. **Malformed rows skipped.** Rows whose length isn't exactly 3 are
   skipped — not an error — on both endpoints. Covered by `TestDetail03`.
4. **Empty result.** The result is nil/empty when the API returns no
   rows. Covered by `TestDetail04`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | yes |
