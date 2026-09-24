# DETAILS — timestamp

1. Quoted RFC3339 strings decode as usual, including fractional-second forms
   (`"2006-01-02T15:04:05.000Z"`). Inferable: doc — the type doc and the bug
   report both name the string format.
2. A bare JSON number decodes as a Unix-seconds timestamp (`1136214245` →
   `2006-01-02T15:04:05Z`). Inferable: doc — the doc comment advertises "Unix
   timestamp" support; seconds granularity is the plain reading.
3. A bare number whose *seconds* interpretation would land after year 3000 is
   instead decoded with millisecond granularity (`1136214245000` → the same
   2006 instant; `1615077308538` → 2021-03-06; `1136214245001` → one
   millisecond after the 2006 instant). Inferable: no — the
   decoded-year-triggered reinterpretation is an arbitrary rule nothing in the
   tree implies.
4. The granularity switch is strict: `32503680000` (year 3000-01-01 exactly)
   still decodes as seconds, and every larger value flips to milliseconds.
   Inferable: no — the exact boundary and its strictness are arbitrary.
5. 11-digit values in the range that decodes to years 2286–3000 (e.g.
   `30000000000` → year 2927) decode as seconds, not milliseconds — the switch
   is on the decoded instant's year, not the digit count or raw magnitude.
   Inferable: no — nothing in the tree implies the window's edges.
6. `0` decodes to the Unix epoch, not zero-time and not an error. Inferable:
   partially — the epoch follows from seconds decoding, but reading `0` as
   "unset" is equally plausible.
7. Quoted non-times (`"asdf"`), quoted digits, and a literal `null` are decode
   errors — the error is returned, not swallowed into zero-time. Inferable:
   partially — erroring is derivable, but silent-zero is an equally reasonable
   choice.
