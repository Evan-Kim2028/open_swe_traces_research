# Details — optsparse

1. `parseDuration` on a STRING does `time.ParseDuration` — error appends a
   `configErr` and returns 0. On a bare int it treats the value as
   SECONDS, appends a `configWarningErr` ("should be converted to a
   duration"), and returns the scaled duration — backward compat, warn
   not fail. Inferable: doc — TestParseWriteDeadline covers both paths.
2. `parseWriteDeadlinePolicy` accepts only "default"|"close"|"retry" and
   falls back to WriteTimeoutPolicyDefault on bad input after appending a
   configErr. Inferable: yes.
3. `parseListen`: `int64` → port with empty host; string →
   `net.SplitHostPort` (so bare "8922" FAILS the split — that config form
   must arrive as an int), port must `Atoi`; any other type → error.
   Inferable: partially — the int-vs-string asymmetry is subtle.
4. `parseURL` trims whitespace then `url.Parse`; errors mention the type.
   `parseURLs` dedupes exact strings (duplicate → warning, not error),
   accumulates per-entry errors WITHOUT aborting the list. Inferable:
   partially.
5. `getStorageSize`: int64 passes through; "" → 0; string must end in a
   K/M/G/T suffix (map to shift 10/20/30/40) — `1K`=1024 … `1T`=2^40; a
   non-numeric prefix or unknown suffix → error. Inferable: yes —
   TestGetStorageSize pins the table.
6. `parseCompression` accepts: string (mode verbatim, validated later),
   bool (true→`chosenModeForOn`, false→`CompressionOff`), or map with keys
   `mode` and any of `rtt_thresholds|thresholds|rtts|rtt` (duration list);
   unknown keys → configErr UNLESS the key token is a used variable.
   Inferable: partially — key synonyms are internal.
7. `trackExplicitVal` lazily allocates the map and sets `m[name]=val` —
   even false is recorded (explicit-vs-default matters). Inferable: yes.
8. All parsers take `token`s and append to `errors`/`warnings` slices
   rather than returning errors — config errors are positional.
   Inferable: yes — signature-level.
