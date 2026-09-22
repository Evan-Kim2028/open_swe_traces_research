# Exported API — statsreshape

`RepositoriesService.ListCodeFrequency` and `ListPunchCard` return raw
`[][]int` from the API reshaped into typed slices:

- Code frequency: `[week_unix, additions, deletions]` →
  `*WeeklyStats{Week *Timestamp, Additions, Deletions *int}`.
- Punch card: `[day, hour, commits]` →
  `*PunchCard{Day, Hour, Commits *int}`.

Rows not exactly length 3 are skipped.
