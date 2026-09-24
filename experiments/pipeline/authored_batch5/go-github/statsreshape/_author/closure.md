# Closure — statsreshape

Package: github (root). File: github/repos_stats.go.
Removed bodies: ListCodeFrequency and ListPunchCard — stubbed to return
`nil` slices, dropping the int-array → struct reshape: code-frequency
`[unix_ts, additions, deletions]` triples become `*WeeklyStats` and
punch-card `[day, hour, commits]` triples become `*PunchCard`. The API
call and raw decode keep working.
Kept: WeeklyStats/PunchCard types, Timestamp, response plumbing.
Tests removed: 2 funcs in github/repos_stats_test.go
(TestRepositoriesService_ListCodeFrequency,
TestRepositoriesService_ListPunchCard).
