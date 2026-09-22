# Contract — msgtrace

Message-trace header lifting plus remote-name and compression helpers.
Every commitment below is covered by a hidden test; every hidden test
maps to a commitment.

## Commitments

1. **Header block parsing.** The block must begin with the status line;
   lines are `key:value` with the key ending at the first colon and the
   value trimmed of spaces and tabs on both sides; keys or values that
   end empty are skipped entirely; a trailing line without a terminator
   is not parsed. Covered by `TestDetail01`.
2. **Key matching.** The trace-destination key matches case-sensitively
   while the traceparent key matches case-insensitively, and the
   original key case is preserved in the output map. Covered by
   `TestDetail02`.
3. **Disabled sentinel.** A trace-destination value equal to the
   disabled sentinel yields no map and a false flag. Covered by
   `TestDetail03`.
4. **Traceparent validity.** The traceparent value counts only when it
   splits into exactly four dash-separated tokens whose fourth token is
   a two-character hex value with the sampled bit set. Covered by
   `TestDetail04`.
5. **Return contract.** The map is nil unless a destination or a
   sampled traceparent was found; the flag is true only when
   traceparent alone lifted the headers; duplicate keys accumulate
   values in order. Covered by `TestDetail05`.
6. **Connection name.** Routers, gateways, and leaf connections report
   their remote name, falling back to the configured name when empty;
   every other kind reports the configured name. Covered by
   `TestDetail06`.
7. **Compression mapping.** An empty accept-encoding yields no
   compression; case-insensitive substring matches on snappy/s2 and
   gzip map to their types; anything else is unsupported. Covered by
   `TestDetail07`.
8. **Sampler.** Out-of-range percentages are treated as 100%; in-range
   values sample at roughly their rate. Covered by `TestDetail08`.
9. **Support check.** Client connections always support tracing; other
   kinds require the remote protocol to be at least the trace protocol
   version. Covered by `TestDetail09`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — the "stop at first colon-less line" edge was not asserted (observed behavior merges rather than stops; not pinnable) |
| TestDetail02 | 2 | doc — asserted exactly |
| TestDetail03 | 3 | partially — (nil,false) asserted |
| TestDetail04 | 4 | doc — asserted exactly |
| TestDetail05 | 5 | yes — asserted exactly |
| TestDetail06 | 6 | yes — asserted exactly |
| TestDetail07 | 7 | partially — substring matching asserted via embedded-substring inputs |
| TestDetail08 | 8 | doc — boundaries asserted deterministically; rate asserted with wide statistical band |
| TestDetail09 | 9 | yes — asserted exactly |
