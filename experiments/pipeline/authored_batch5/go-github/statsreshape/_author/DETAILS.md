# DETAILS — statsreshape

1. Each code-frequency `[unix_ts, additions, deletions]` row becomes a
   `*WeeklyStats` — element 0 → `Week` (as a `*Timestamp` of that unix
   time), 1 → `Additions`, 2 → `Deletions`. Inferable: partially — the
   reshape is derivable from the API doc; the field order mapping is
   the actual commitment.
2. Each punch-card `[day, hour, commits]` row becomes a `*PunchCard`
   — 0 → `Day`, 1 → `Hour`, 2 → `Commits`. Inferable: partially.
3. Rows whose length isn't exactly 3 are skipped (not an error).
   Inferable: partially — lenient skip is an arbitrary choice; a solver
   could equally error.
4. The result is `nil`/empty when the API returns no rows.
   Inferable: yes.
