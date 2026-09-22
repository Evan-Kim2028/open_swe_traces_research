# Contract — cgfile

Reader for the on-disk commit-graph file (`CGPH`), its table of contents,
fanout, commit-data and edge/generation chunks, plus chained-graph
coalescing. Every commitment below is covered by a hidden test; every hidden
test maps to a commitment.

## Commitments

1. **Header discipline.** Opening a file checks the `CGPH` signature, a
   version byte of exactly 1, and a hash-function byte matching the reader's
   object format; a wrong version reports the unsupported-version error and
   a wrong hash byte the unsupported-hash error. Covered by `TestDetail01`.
2. **Size precheck.** When the reader can report a size, a file too small to
   hold the header, the declared chunk table plus terminator, the fanout and
   the hash trailer fails on open; when the reader cannot report a size the
   precheck is skipped and a well-formed file still opens. Covered by
   `TestDetail02`.
3. **TOC walk.** The table of contents is walked exactly the declared number
   of chunks; offsets must be monotonically non-decreasing and below the
   file size minus the trailer — violations fail the open. Covered by
   `TestDetail03`.
4. **Chunk-id validity (shape).** A duplicate chunk id — or a zero id before
   the declared count ends — is malformed: the open returns an error.
   Covered by `TestDetail04`.
5. **Terminator.** A single zero-id terminator entry must follow the table;
   a non-zero entry in that position fails the open. Covered by
   `TestDetail05`.
6. **Cardinality at open.** Chunk sizes are checked against the
   fanout-derived commit count at open time — a file whose lookup chunk is
   too short for the declared commit count fails on open. Covered by
   `TestDetail06`.
7. **Lookup order.** Hash lookup buckets by the first hash byte through the
   fanout then binary-searches the OID table; a miss falls through to the
   parent index, and a hash absent everywhere reports not-found. Covered by
   `TestDetail07`.
8. **Parent delegation.** Indexes below the parent's hash count delegate to
   the parent chain — the parent's global numbering is preserved and the
   local fanout total is not consulted for those indexes. Covered by
   `TestDetail08`.
9. **Octopus edges.** A merge with more than two parents reads extra parents
   from the edge-list chunk until a terminator bit, bounded by the chunk's
   derived entry count — an unterminated walk reports an error. Covered by
   `TestDetail09`.
10. **GenerationV2.** The corrected commit date combines the commit time and
    the per-commit generation datum, and when the datum's high bit is set
    the low bits index the overflow chunk — encoded values, including ones
    requiring the overflow path, read back exactly. Covered by
    `TestDetail10`.
11. **Chain file (shape).** The chain file is a newline-separated list of
    graph hashes oldest-to-newest, each line validated as a full object id;
    a malformed line fails the whole read, and a final line with no trailing
    newline is silently dropped rather than accepted. Covered by
    `TestDetail11`.
12. **Open order.** The chain-or-file open prefers the single graph file and
    falls back to the chain only when the file cannot be opened — a file
    that opens but fails to parse is a hard error, not a fallback. Covered
    by `TestDetail12`.
13. **Chained close (shape).** Closing a chained index also closes the
    parent; the parent's close error is reported only when the reader itself
    closed cleanly — a reader close error takes precedence. Covered by
    `TestDetail13`.
14. **GenV2 AND-fold (shape).** GenerationV2 availability is the AND of this
    file's chunk presence with the parent's own flag — a chain advertises it
    only when every link has it. Covered by `TestDetail14`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | doc |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | no — shape only |
| TestDetail05 | 5 | doc |
| TestDetail06 | 6 | doc |
| TestDetail07 | 7 | partially |
| TestDetail08 | 8 | partially |
| TestDetail09 | 9 | partially |
| TestDetail10 | 10 | partially |
| TestDetail11 | 11 | no — shape only |
| TestDetail12 | 12 | partially |
| TestDetail13 | 13 | no — shape only |
| TestDetail14 | 14 | no — shape only |
