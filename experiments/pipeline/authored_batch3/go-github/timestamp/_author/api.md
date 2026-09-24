# Exported API — timestamp

`Timestamp` is the time type used across the whole client: virtually every API
object that carries a time field embeds it, so its JSON decode is exercised by
every endpoint that returns a response body.

- `Timestamp` — wraps `time.Time`; all exported `time.Time` methods work on it.
- `(*Timestamp).UnmarshalJSON` — the custom JSON decoder (excised).
- `(Timestamp).Equal`, `(Timestamp).String`, `(*Timestamp).GetTime` — unaffected.
- Marshal side is untouched: a `Timestamp` always marshals as a quoted RFC3339
  string.
