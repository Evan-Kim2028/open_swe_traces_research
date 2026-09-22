# Details — msgtrace

1. `genHeaderMapIfTraceHeadersPresent` requires the `NATS/1.0\r\n` status
   line; otherwise nil. Lines are `key:value` — key ends at the first `:`,
   value is trimmed of spaces AND tabs on both sides; a key or value that
   ends empty is skipped entirely (not even recorded). Parse stops at the
   first line lacking `:` or CRLF. Inferable: partially — the whole table
   test pins these edges.
2. `MsgTraceDest` matching is case-SENSITIVE; `traceparent` matching is
   case-INSENSITIVE (`EqualFold`), and the original key case is preserved
   in the output map (no lowercase rewrite since v2.14). Inferable: doc —
   comment + tests.
3. A `MsgTraceDest` value equal to the disabled sentinel short-circuits to
   `(nil, false)` — no map at all. Inferable: partially — the constant is
   visible.
4. `traceparent` counts as "found" only when the value splits into exactly
   4 `-`-separated tokens, the 4th is 2 chars, hex-parses, and has bit
   `0x1` set (the sampled flag). A malformed or unsampled traceparent does
   not lift the headers. Inferable: doc — W3C format + test table.
5. Return: nil unless dest OR a sampled traceparent was found; the bool is
   true only when ONLY traceparent was found ("external"). Duplicate keys
   accumulate values in order. Inferable: yes — test.
6. `getConnName`: ROUTER→`c.route.remoteName`, GATEWAY→`c.gw.remoteName`,
   LEAF→`c.leaf.remoteServer`, each falling through to `c.opts.Name` when
   empty; CLIENT and everything else → `c.opts.Name`. Inferable: yes —
   test.
7. `getCompressionType`: empty → `noCompression`; case-insensitive
   substring match — "snappy" or "s2" → snappy, "gzip" → gzip, else
   `unsupportedCompression`. Inferable: partially — substring (not token)
   matching is a choice.
8. `sample`: out-of-range [1..99] is treated as 100%; otherwise
   `rand.Int32N(100) <= sampling`. Inferable: doc — comment.
9. `msgTraceSupport`: CLIENT kind always true; otherwise
   `c.opts.Protocol >= MsgTraceProto`. Inferable: yes.
