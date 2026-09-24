# Details — gceurl

1. `BuildURL` defaults `Version` to `v1` and always emits trailing `/` between segments;
   `global`, `regions` and `zones` are independent optional segments. Inferable: yes — it's
   the inverse of the parser.
2. `ParseGoogleCloudURL` rejects any scheme/host other than `https://www.googleapis.com`,
   any service other than `compute`, and any version other than `v1`/`beta`. Inferable:
   partially — the whitelist is arbitrary.
3. `regions` is only recognised when at least 2 tokens follow it (else falls through to
   default = treated as a TYPE token) — a subtle quirk vs `zones`/`projects` which always
   consume the next token. Inferable: no — positional quirk.
4. After `TYPE/NAME` the parser REQUIRES end-of-input; trailing segments error.
   Inferable: yes — strict round-trip.
5. `EncodeGCELabel` operates on BYTES (not runes): only `[0-9a-z]` pass unescaped —
   uppercase letters are escaped (`A` → `-41`). Escape is `-` + two lowercase hex digits.
   Inferable: no — charset and escape format are arbitrary.
6. `DecodeGCELabel` maps `-`→`%` then `url.QueryUnescape`; malformed escapes error (the
   original label is echoed in the error). Inferable: partially.
7. `TagForRole` composes the role prefix with `ClusterPrefixedName` at 63 chars.
   Inferable: partially — the 63 cap and prefix string are conventions.
