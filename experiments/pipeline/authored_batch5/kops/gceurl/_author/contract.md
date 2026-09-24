# Contract — gceurl

GCE compute URL construction/parsing and label encoding. Every commitment
below is covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **`BuildURL`.** Composes `https://www.googleapis.com/compute/<version|v1>/`
   plus optional `projects/P/`, `global/`, `regions/R/`, `zones/Z/` segments
   and `TYPE/NAME`; `Version` defaults to `v1`. Round-trips through the
   parser. Covered by `TestDetail01`.
2. **Strict parse.** Only `https://www.googleapis.com`, the `compute`
   service, and `v1`/`beta` versions are accepted. Covered by
   `TestDetail02`.
3. **Segment keywords (shape).** `regions`/`zones`/`projects` consume a
   following value token into `Region`/`Zone`/`Project`. The under-fed
   `regions` fall-through is exercised only for shape: no panic, and a
   successful parse cannot claim a `Region`. Covered by `TestDetail03`.
4. **Terminal TYPE/NAME.** After `TYPE/NAME` the parser requires
   end-of-input — trailing segments or a missing name error. Covered by
   `TestDetail04`.
5. **`EncodeGCELabel` (shape).** Per the API doc, lowercase alnum bytes pass
   through and every other byte becomes `-` + two lowercase hex digits —
   byte-wise (UTF-8 input produces one escape per byte). Covered by
   `TestDetail05`.
6. **`DecodeGCELabel`.** Inverts `EncodeGCELabel`; malformed escapes return
   an error that echoes the input label. Covered by `TestDetail06`.
7. **`TagForRole`.** Equals `ClusterPrefixedName` applied to the role-prefixed
   lowercase role name at the documented length cap; never exceeds the cap.
   Covered by `TestDetail07`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | partially — whitelist asserted per api.md |
| TestDetail03 | 3 | no — quirk asserted shape-only (no Region on under-fed input) |
| TestDetail04 | 4 | yes |
| TestDetail05 | 5 | no — asserted per api.md rule + documented example |
| TestDetail06 | 6 | partially — round-trip + error-echoes-input |
| TestDetail07 | 7 | partially — asserted via ClusterPrefixedName equivalence per api.md |
