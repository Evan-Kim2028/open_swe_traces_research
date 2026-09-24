# Contract — mondecode

HTTP monitoring request decoding and display helpers. Every commitment
below is covered by a hidden test; every hidden test maps to a
commitment.

## Commitments

1. **Optional scalar params.** An absent query parameter is not an
   error — the zero value returns with nil error. A parse failure writes
   an HTTP 400 whose body names the offending parameter and returns the
   parse error. Covered by `TestDetail01`.
2. **Connection state.** The state parameter defaults to open;
   open/closed/any/all match case-insensitively with any and all
   synonymous; an unrecognized value writes 400, returns an error, and
   yields the zero state. Covered by `TestDetail02`.
3. **Subscriptions flag.** `subs=detail` (any case) selects the detail
   form and bypasses boolean decoding; every other value is decoded as a
   boolean, so an invalid value produces the 400 path. Covered by
   `TestDetail03`.
4. **Uptime rendering.** The rendered duration starts at the largest
   non-zero unit and emits every lower unit once one appears, with years
   computed as whole days divided by 365. Covered by `TestDetail04`.
5. **Bearer redaction.** Empty input stays empty; input that decodes to
   a bearer-token user claim redacts to empty; undecodable input and
   decodable non-bearer claims return the original string. Covered by
   `TestDetail05`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc — zero-value/nil and 400-with-param-name; the per-type label word is not pinned |
| TestDetail02 | 2 | partially — case-insensitive map and synonym; exact error text not pinned |
| TestDetail03 | 3 | yes |
| TestDetail04 | 4 | doc — exact strings |
| TestDetail05 | 5 | partially — redact-vs-keep direction |
