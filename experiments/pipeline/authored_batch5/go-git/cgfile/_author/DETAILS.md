# Details — cgfile

1. The header check demands the `CGPH` signature, version byte exactly 1, and
   a hash-function byte matching the reader's object format — each failure
   maps to its own error var, not a shared one. Inferable: partially — the
   error vars and signature const are visible, the byte layout is the detail.
2. A whole-file size precheck requires room for the header, the declared
   chunk table plus its terminator, the fanout, and the hash trailer — and is
   skipped entirely when the reader cannot report a size. Inferable: doc —
   the verifyFileSize doc comment describes it.
3. The table of contents is walked exactly `numChunks` times; every chunk
   offset must be monotonically non-decreasing and below fileSize-minus-
   trailer — Inferable: partially — the doc comment gives the walk, the
   ordering guard is the detail.
4. A duplicate chunk id — known or unknown — is malformed, and so is a zero
   id appearing before the declared count ends. Inferable: no.
5. A single zero-id terminator entry must follow the table; each chunk's byte
   length is derived from the NEXT table offset (or the terminator for the
   last). Inferable: doc — the readChunkHeaders comment spells it out.
6. Fanout, OID-lookup, commit-data and (when present) generation chunks are
   cardinality-checked against the fanout-derived commit count at open time —
   a truncated file fails on open, not mid-walk. Inferable: doc —
   verifyChunkSizes' comment lists the checks.
7. Hash lookup buckets by the first hash byte through the fanout — byte 0
   starts at index 0, others at fanout[byte-1] — then binary-searches the OID
   table; a miss falls through to the parent index before reporting not-
   found. Inferable: partially.
8. Indexes below the parent's hash count delegate to the parent chain — the
   parent's global numbering is preserved, the local fanout total is never
   consulted for those. Inferable: partially.
9. An octopus merge (high bit on the second parent slot) reads extra parents
   from the edge-list chunk until a terminator bit, bounded by the chunk's
   derived entry count — an out-of-range or unterminated walk is malformed.
   Inferable: partially — the chunk type names are visible.
10. GenerationV2 is commit-time (low 34 bits of the packed field) plus the
    per-commit generation datum — and when that datum's high bit is set the
    low 31 bits index the OVERFLOW chunk for a corrected 64-bit timestamp,
    bounded by that chunk's derived size. Inferable: partially.
11. A chain file is a newline-separated list of graph hashes oldest-to-newest,
    each line validated as a full object id — a malformed line fails the whole
    read, and a final line with no trailing newline is silently DROPPED rather
    than accepted. Inferable: no — the trailing-line loss is an arbitrary edge.
12. The chain-or-file open prefers the single graph file and falls back to the
    chain only when the file cannot be OPENED — a file that opens but fails to
    parse is a hard error, not a fallback. Inferable: partially — the doc
    comment names the order, the error-path split is the detail.
13. Closing a chained index also closes the parent, and the parent's close
    error is reported only when the reader itself closed cleanly. Inferable:
    no.
14. GenerationV2 availability is reported as the AND of this file's chunk
    presence with the parent's own flag — a chain advertises it only when
    every link has it. Inferable: no.
