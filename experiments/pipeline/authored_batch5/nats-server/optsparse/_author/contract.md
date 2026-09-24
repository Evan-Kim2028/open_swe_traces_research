# Contract — optsparse

Scalar config-value parsers invoked by the config dispatcher. Every
commitment below is covered by a hidden test; every hidden test maps to
a commitment.

## Commitments

1. **Duration parsing.** A string is parsed as a Go duration; a bad
   string appends a config error and returns zero. A bare integer is
   treated as seconds, appends a warning, and returns the scaled
   duration. Covered by `TestDetail01`.
2. **Write-deadline policy.** Only `default`, `close`, and `retry` are
   accepted; anything else appends a config error and falls back to the
   default policy. Covered by `TestDetail02`.
3. **Listen spec.** An integer is a port with empty host; a string is
   split as `host:port` so a bare port string fails; the port must be
   numeric; other types error. Covered by `TestDetail03`.
4. **URL lists.** A URL is whitespace-trimmed then parsed. A list
   dedupes exact strings with a warning (not an error) and accumulates
   per-entry errors without aborting. Covered by `TestDetail04`.
5. **Storage size.** An integer passes through; an empty string is
   zero; a string must end in a K/M/G/T suffix mapping to powers of
   two; a non-numeric prefix or unknown suffix errors. Covered by
   `TestDetail05`.
6. **Compression spec.** A string sets the mode verbatim; a bool picks
   the chosen-for-on mode or off; a map accepts `mode` and an
   rtt-threshold synonym key with a duration list; unknown keys error
   unless the value is a used variable. Covered by `TestDetail06`.
7. **Explicit-value tracking.** The tracking map is lazily allocated
   and records every explicitly-set boolean, including false. Covered
   by `TestDetail07`.
8. **Error plumbing.** Parsers append to errors/warnings slices keyed
   by the token's position rather than returning errors. Covered by
   `TestDetail08`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc — both paths asserted |
| TestDetail02 | 2 | yes — asserted exactly |
| TestDetail03 | 3 | partially — int-vs-string asymmetry asserted |
| TestDetail04 | 4 | partially — trim, dedupe-warning, accumulate-not-abort |
| TestDetail05 | 5 | yes — pinned table plus error cases |
| TestDetail06 | 6 | partially — all four key synonyms and the used-variable exemption |
| TestDetail07 | 7 | yes — asserted exactly |
| TestDetail08 | 8 | yes — position-bearing appended error asserted via the duration path |
