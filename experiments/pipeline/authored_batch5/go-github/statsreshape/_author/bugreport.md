# Bug report — statsreshape

`ListCodeFrequency` and `ListPunchCard` return empty results even when
the API sent data: the raw `[][]int` rows are never reshaped into
`*WeeklyStats` / `*PunchCard`.

Expected: each 3-element row becomes one typed struct — week timestamp +
additions + deletions, or day + hour + commits.

Got: nil slices regardless of what the API returned.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
