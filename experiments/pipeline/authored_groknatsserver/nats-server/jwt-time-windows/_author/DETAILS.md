1. Nil claims yield `(false, 0)`. Inferable: yes
2. A non-nil claims with an empty `Times` slice yields `(true, 0)`. Inferable: yes
3. The location is `time.Local` unless `claims.Locale` is set, in which case `time.LoadLocation` is used and `now` is converted into that location. Inferable: yes
4. A locale that fails to load yields `(false, 0)`. Inferable: yes
5. Each range's `Start` and `End` are parsed with the clock layout `15:04:05` in that location. Inferable: no
6. Start and end are then re-seated onto `now`'s calendar date in that location, with nanoseconds zeroed. Inferable: yes
7. A range that does not wrap midnight matches only when `start.Before(now) && end.After(now)` (endpoints exclusive). Inferable: no
8. Remaining time on a match is `end.Sub(now)`. Inferable: doc
9. A range wraps midnight when `start.After(end)`. Inferable: doc
10. After midnight in a wrapping range, `end.After(now)` is enough to match, with remaining `end.Sub(now)`. Inferable: doc
11. Before midnight in a wrapping range, `end` is advanced by one calendar day and then the exclusive `start.Before(now) && end.After(now)` test is applied. Inferable: doc
12. When several ranges match, the returned remaining duration is the maximum among them. Inferable: partially
13. If no range matches, the result is `(false, 0)`. Inferable: yes
14. A start or end string that fails to parse yields `(false, 0)` for the whole call. Inferable: yes
