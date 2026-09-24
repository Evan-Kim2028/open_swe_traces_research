# Details — mondecode

1. `decodeBool`/`decodeUint64`/`decodeInt`: absent param is NOT an error —
   returns the zero value with nil error. Parse failure writes HTTP 400
   with `Error decoding <type> for '<param>': <err>` AND returns the
   error. Inferable: doc — the write-then-return pattern is uniform and
   visible at call sites.
2. `decodeState`: `state` absent → `ConnOpen`; values open/closed/any/all
   matched case-insensitively (any and all are synonyms); anything else →
   400 + error, return value 0. Inferable: partially — the any/all
   synonym is only in source.
3. `decodeSubs`: `subs=detail` (case-insensitive) sets subsDet and skips
   bool decoding entirely; any other value goes through decodeBool
   semantics (so `subs=garbage` → 400). Inferable: yes — handler usage.
4. `myUptime` renders the largest non-zero unit first and always emits all
   lower units once a unit has appeared: `22s`, `4m22s`, `4h4m22s`,
   `32d4h4m22s`, `22y32d4h4m22s`. Years divide days by 365; units are
   whole-second truncations of the duration. Inferable: doc — test pins
   exact strings.
5. `redactBearerJWT`: empty input → empty; otherwise decodes user claims
   and returns empty ONLY when decode succeeds AND `BearerToken` is set;
   undecodable or non-bearer input returns the original string.
   Inferable: partially — the "decode fails → keep original" direction is
   a deliberate redaction choice.
