# Exported API — hexdump

Package `internal/logutil` (module `example.internal/kvstore/v2`).

- `Hex(msg proto.Message) fmt.Stringer` — wraps any proto message so
  `Sprintf("%v"/"%s")` yields the brace-form with byte fields in hex.
- `hexStringer.String() string` — the backing `String()`.
- `prettyPrint(io.Writer, reflect.Value)` — recursive walker (unexported;
  reached only through `Hex`).

Callers: logging sites that print requests/regions/peers, e.g.
`logutil.Hex(region.Meta)`.
