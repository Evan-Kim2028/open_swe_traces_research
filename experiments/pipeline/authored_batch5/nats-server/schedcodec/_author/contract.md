# Contract — schedcodec

Binary snapshot codec for pending message schedules plus the
`Nats-Schedule` pattern parser. Every commitment below is covered by a
hidden test; every hidden test maps to a commitment.

## Commitments

1. **Encode layout.** Version byte 1, little-endian entry count,
   little-endian high-sequence stamp; per entry a u16 subject length,
   subject bytes, a signed varint timestamp, and a uvarint sequence.
   Covered by `TestDetail01`.
2. **Decode errors and bookkeeping.** A buffer under the header size is
   a short-buffer error; a non-1 version byte is the invalid-version
   error; any truncation past the header is unexpected-EOF; a valid
   buffer populates the schedules, sequence index, and timer wheel and
   returns the stamp. Covered by `TestDetail02`.
3. **Empty pattern.** An empty schedule pattern is a valid no-op
   returning the zero time. Covered by `TestDetail03`.
4. **@at / @every.** `@at` parses an RFC3339 instant as a one-shot;
   `@every` parses an interval of at least one second as repeating;
   neither accepts a non-nil location. Covered by `TestDetail04`.
5. **Predefined aliases.** `@yearly`/`@annually`/`@monthly`/`@weekly`/
   `@daily`/`@midnight`/`@hourly` expand to their six-field cron
   equivalents; anything else parses as a cron spec and failure is
   invalid. Covered by `TestDetail05`.
6. **Past-fire catch-up.** A computed next fire in the past is skipped
   forward — `@every` re-arms from now rounded to the second plus one
   interval — and remains repeating. Covered by `TestDetail06`.
7. **Anchoring.** `@every` anchors on the supplied timestamp rounded to
   the second plus the interval. Covered by `TestDetail07`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — field order/widths asserted; entry order left unordered (map) |
| TestDetail02 | 2 | yes — exact error identities asserted per truncation |
| TestDetail03 | 3 | partially — (zero, false, true) asserted |
| TestDetail04 | 4 | doc — asserted including loc rejection |
| TestDetail05 | 5 | partially — asserted via observable next-fire instants, not the internal table |
| TestDetail06 | 6 | partially — bump-forward and repeat flag asserted with a 1s tolerance band |
| TestDetail07 | 7 | partially — anchor arithmetic asserted exactly |
