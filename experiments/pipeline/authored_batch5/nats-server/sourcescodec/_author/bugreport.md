# Bug report — sourcescodec

**Title:** Sourced-message restart state lost: sources.db decode
accepts torn buffers and the source header mis-parses the new format.

**Symptoms:**
- A truncated `sources.db` (power failure mid-write) decodes without
  error and seeds a partial sources map — on restart the stream
  re-sources already-ingested messages, producing duplicates.
- `streamAndSeq` accepts a 3-field header as v2, so `"a b c"` decodes
  to iname `a c <empty>` and seq from a non-seq token — the source
  progress index lands under a garbage key.
- `genSourceHeader` reads the stream sequence from the wrong
  `$JS.ACK` token, stamping headers with the delivery count, so the
  decode side records a wildly wrong `Seq`.
- ident is never copied, so upstream-identity dedup silently breaks
  across restarts.

**Reproduction:** store sourced messages, truncate `sources.db` to a
partial entry, restart — gold warns and discards the file, the
defective decode silently accepts it.

**Expected:** strict uvarint/length/EOF checks per field, the
2-or-≥4 arity rule, correct ack token offsets, ident copy.
