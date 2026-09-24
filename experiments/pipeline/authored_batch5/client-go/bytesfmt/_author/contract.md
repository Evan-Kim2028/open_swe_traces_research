# Contract (L2) — bytesfmt

`FormatBytes` renders a byte count with pruned precision; at or below the
smallest unit boundary it produces exactly what `BytesToString` produces
(the raw count, unscaled), and above it the rendering is scaled —
`FormatBytes(1025)` is visibly shorter than the raw-float `BytesToString`
rendering of the same value, and the raw count `2048` no longer appears in
`FormatBytes(2048)`. `BytesToString` keeps a value that works out to
exactly one unit in the Bytes row — `BytesToString(1024)` still prints the
raw count — and scales larger values up.
`CompatibleParseGCTime` parses the stored gc_worker time format; a trailing
zone abbreviation is tolerated by dropping the last space-separated field
and reparsing, and unparseable input returns an error.
`EncodeToString` hex-encodes to a `[]byte`; `HexRegionKey` is the uppercase
hex form and `HexRegionKeyStr` its string form. `ToUpperASCIIInplace`
uppercases only `a`–`z` in place and returns the same slice.
`String` is a zero-copy `[]byte`→`string` view: mutating the bytes is
visible through the result.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | `FormatBytes(n)` equals `BytesToString(n)` at or below the Bytes threshold and renders the raw count there |
| `TestDetail02` | `FormatBytes` prunes precision (differs from `BytesToString` on a scaled value) and scales larger inputs into larger units |
| `TestDetail03` | `BytesToString(1024)` stays in the Bytes row while `BytesToString(2048)` scales |
| `TestDetail04` | `CompatibleParseGCTime` parses the stored format, tolerates a trailing zone abbreviation, and errors on junk |
| `TestDetail05` | `EncodeToString`/`HexRegionKey`/`HexRegionKeyStr` produce lowercase []byte hex / uppercase hex / uppercase string |
| `TestDetail06` | `ToUpperASCIIInplace` uppercases only `a`–`z` in place |
| `TestDetail07` | `String` returns a zero-copy view of the bytes |
