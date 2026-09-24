# Contract — mqtttopics

Bidirectional MQTT-topic ↔ subject conversion and topic validation.
Every commitment below is covered by a hidden test; every hidden test
maps to a commitment.

## Commitments

1. **Forward conversion.** `/` becomes `.`, `.` becomes `//`, `+`
   becomes `*`, `#` becomes `>`; a leading or trailing `/` and doubled
   `//` keep their empty levels via `/.` insertions. Covered by
   `TestDetail01`.
2. **Reverse conversion.** The reverse mapping collapses `//` to `.`
   and `/.` pairs to `/`, consuming both characters in a left-to-right
   scan. Covered by `TestDetail02`.
3. **Unsupported characters.** Whitespace and DEL anywhere in a topic or
   filter reject with the unsupported-characters error, in both the
   publish and filter directions. Covered by `TestDetail03`.
4. **Wildcard rules.** Publish topics reject any wildcard with an error
   naming the topic; filters convert `+`/`#` at any position, including
   positions MQTT would consider invalid. Covered by `TestDetail04`.
5. **No-allocation fast path.** Input needing no conversion is returned
   as the same backing array. Covered by `TestDetail05`.
6. **Level-up subscription.** A subject of at least three characters
   ending in `.>` needs an extra subscription one level up; anything
   else does not. Covered by `TestDetail06`.
7. **Validation scope.** Topic/string validation rejects only embedded
   NUL and invalid UTF-8; empty values and wildcard spellings pass.
   Covered by `TestDetail07`.
8. **Reserved subscriptions.** A subscription whose filter is the
   catch-all or begins with the first-level-wildcard spelling is
   reserved; deliveries of `$`-prefixed subjects are dropped for
   reserved subscriptions only. Covered by `TestDetail08`.
9. **Sparkplug birth/death topics (shape).** Topics under the sparkplug
   namespace with three or four parts after any certificates prefix
   parse as birth/death only for the birth and death type tokens; other
   type tokens and wrong part counts match neither, and the
   certificates prefix sets the certificate flag. Covered by
   `TestDetail09`.
10. **Death-timestamp rewrite (shape).** The timestamp rewrite returns
    the original buffer on a scan error, substitutes the timestamp
    field when present while preserving other fields, and appends a
    timestamp field when none was present. Covered by `TestDetail10`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — documented mappings plus empty-level preservation |
| TestDetail02 | 2 | no — asserted as round-trip pairs from the visible comment rules |
| TestDetail03 | 3 | doc — representative members of the rejected set |
| TestDetail04 | 4 | partially — rejection names the topic; any-position conversion |
| TestDetail05 | 5 | doc — backing-array identity |
| TestDetail06 | 6 | partially — length and suffix rule |
| TestDetail07 | 7 | doc |
| TestDetail08 | 8 | partially — reserved spellings and the $-drop gate; the `#` spelling is not asserted (subject-domain only) |
| TestDetail09 | 9 | no — shape only: which type tokens count, part-count window, certificate flag |
| TestDetail10 | 10 | no — shape only: error→identity, substitution preserves other fields, missing→appended |
