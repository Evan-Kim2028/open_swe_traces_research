# Contract — parse-msg-schedule

Hidden suite: `tests/hidden/server/parse_msg_schedule_bb_test.go`
(package `server`, in-package). One `TestDetailNN` per DETAILS.md line.
Time-based assertions are windowed against a `time.Now()` bracket taken
around the call, or use a future `ts` for determinism.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | no | The empty pattern yields `(zero time, false, true)` — the committed triple, asserted as-is. |
| TestDetail02 | 2 | doc | `"@at 2030-06-15T10:00:00Z"` yields that instant, `repeating=false`, `ok=true`. |
| TestDetail03 | 3 | doc | `"@at …"` with a non-nil location is illegal (`ok=false`). |
| TestDetail04 | 4 | yes | Malformed `@at` remainders (`"garbage"`, out-of-range RFC3339 fields, empty) all yield `(zero, false, false)`. |
| TestDetail05 | 5 | doc | `"@every 1m"` is repeating, with next = `Unix(0,ts).UTC().Round(Second).Add(1m)` for a future `ts`. |
| TestDetail06 | 6 | doc | `"@every 1m"` with a non-nil location is illegal. |
| TestDetail07 | 7 | no | `@every` intervals under one second are rejected (`999ms`, `500ms`, `1ns`); `1s` is accepted — the floor is asserted as the committed boundary. |
| TestDetail08 | 8 | doc | For a future `ts`, next equals `Unix(0,ts).UTC().Round(Second).Add(dur)` exactly; for a past `ts`, next is recomputed from now and lands in the `now.Round(Second)+dur` bracket. |
| TestDetail09 | 9 | no | `@yearly` and `@annually` are repeating; next is the first second of a January 1 within the coming year. (Shape: calendar boundary, strictly after now.) |
| TestDetail10 | 10 | no | `@monthly` is repeating; next is the first second of a 1st-of-month within the coming month. |
| TestDetail11 | 11 | no | `@weekly` is repeating; next is the first second of a Sunday within the coming week. |
| TestDetail12 | 12 | no | `@daily` and `@midnight` are repeating; next is midnight within the coming day. |
| TestDetail13 | 13 | no | `@hourly` is repeating; next is the top of an hour within the coming hour. |
| TestDetail14 | 14 | yes | A six-field cron pattern (`"0 0 12 * * *"`) dispatches and yields a noon next time; five-field and garbage patterns are illegal, as is an unknown `@` form. |
| TestDetail15 | 15 | doc | An every-second cron with a past `ts` yields a next recomputed from now — inside the call's `now.Round(Second)` bracket. |
| TestDetail16 | 16 | yes | Successful `@every` and cron results report `repeating=true`. |

Refusals/softening: lines 9–13 are `Inferable: no` — each alias is asserted
only as the calendar shape it commits to (which boundary instant, repeating,
strictly after now, within one period), never as a specific expansion string
or internal representation.
